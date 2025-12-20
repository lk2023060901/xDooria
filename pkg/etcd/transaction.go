package etcd

import (
	"context"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
)

// Transaction 事务客户端
type Transaction struct {
	client *Client
}

// newTransaction 创建 Transaction 客户端
func newTransaction(client *Client) *Transaction {
	return &Transaction{
		client: client,
	}
}

// Txn 创建事务
func (t *Transaction) Txn(ctx context.Context) *Txn {
	return &Txn{
		txn: t.client.rawClient().Txn(ctx),
	}
}

// Txn 事务封装
type Txn struct {
	txn clientv3.Txn
}

// If 设置条件
func (t *Txn) If(cmps ...clientv3.Cmp) *Txn {
	t.txn = t.txn.If(cmps...)
	return t
}

// Then 设置成功时的操作
func (t *Txn) Then(ops ...clientv3.Op) *Txn {
	t.txn = t.txn.Then(ops...)
	return t
}

// Else 设置失败时的操作
func (t *Txn) Else(ops ...clientv3.Op) *Txn {
	t.txn = t.txn.Else(ops...)
	return t
}

// Commit 提交事务
func (t *Txn) Commit() (*TxnResponse, error) {
	resp, err := t.txn.Commit()
	if err != nil {
		return nil, err
	}

	return &TxnResponse{
		Succeeded: resp.Succeeded,
		Responses: convertResponses(resp.Responses),
	}, nil
}

// convertResponses 转换响应
func convertResponses(resps []*etcdserverpb.ResponseOp) []interface{} {
	result := make([]interface{}, 0, len(resps))
	for _, resp := range resps {
		switch {
		case resp.GetResponseRange() != nil:
			// Get 响应
			r := resp.GetResponseRange()
			kvs := make([]*KeyValue, 0, len(r.Kvs))
			for _, kv := range r.Kvs {
				kvs = append(kvs, kvToKeyValue(kv))
			}
			result = append(result, kvs)

		case resp.GetResponsePut() != nil:
			// Put 响应
			result = append(result, resp.GetResponsePut())

		case resp.GetResponseDeleteRange() != nil:
			// Delete 响应
			result = append(result, resp.GetResponseDeleteRange().Deleted)

		case resp.GetResponseTxn() != nil:
			// Txn 响应
			result = append(result, resp.GetResponseTxn())
		}
	}
	return result
}

// --- 便捷的事务操作 ---

// CompareAndSwapTxn 比较并交换事务
func (t *Transaction) CompareAndSwapTxn(ctx context.Context, key, oldValue, newValue string) (bool, error) {
	resp, err := t.Txn(ctx).
		If(clientv3.Compare(clientv3.Value(key), "=", oldValue)).
		Then(clientv3.OpPut(key, newValue)).
		Commit()

	if err != nil {
		return false, err
	}

	return resp.Succeeded, nil
}

// CompareVersionAndPut 比较版本并更新
func (t *Transaction) CompareVersionAndPut(ctx context.Context, key string, version int64, value string) (bool, error) {
	resp, err := t.Txn(ctx).
		If(clientv3.Compare(clientv3.Version(key), "=", version)).
		Then(clientv3.OpPut(key, value)).
		Commit()

	if err != nil {
		return false, err
	}

	return resp.Succeeded, nil
}

// CompareAndDelete 比较并删除
func (t *Transaction) CompareAndDelete(ctx context.Context, key, value string) (bool, error) {
	resp, err := t.Txn(ctx).
		If(clientv3.Compare(clientv3.Value(key), "=", value)).
		Then(clientv3.OpDelete(key)).
		Commit()

	if err != nil {
		return false, err
	}

	return resp.Succeeded, nil
}

// AtomicIncrement 原子递增
func (t *Transaction) AtomicIncrement(ctx context.Context, key string, delta int64) (int64, error) {
	for {
		// 获取当前值
		kv := clientv3.NewKV(t.client.rawClient())
		getResp, err := kv.Get(ctx, key)
		if err != nil {
			return 0, err
		}

		var currentValue int64
		var currentVersion int64 = 0

		if len(getResp.Kvs) > 0 {
			// 解析当前值
			currentValue = parseInt64(getResp.Kvs[0].Value)
			currentVersion = getResp.Kvs[0].Version
		}

		newValue := currentValue + delta

		// 使用事务更新
		var succeeded bool
		if currentVersion == 0 {
			// 键不存在，使用 CREATE
			resp, err := t.Txn(ctx).
				If(clientv3.Compare(clientv3.Version(key), "=", 0)).
				Then(clientv3.OpPut(key, int64ToString(newValue))).
				Commit()
			if err != nil {
				return 0, err
			}
			succeeded = resp.Succeeded
		} else {
			// 键存在，比较版本
			resp, err := t.Txn(ctx).
				If(clientv3.Compare(clientv3.Version(key), "=", currentVersion)).
				Then(clientv3.OpPut(key, int64ToString(newValue))).
				Commit()
			if err != nil {
				return 0, err
			}
			succeeded = resp.Succeeded
		}

		if succeeded {
			return newValue, nil
		}

		// 失败重试
	}
}

// --- 辅助函数 ---

func parseInt64(b []byte) int64 {
	var result int64
	for _, v := range b {
		result = result*10 + int64(v-'0')
	}
	return result
}

func int64ToString(n int64) string {
	if n == 0 {
		return "0"
	}

	negative := n < 0
	if negative {
		n = -n
	}

	var digits []byte
	for n > 0 {
		digits = append([]byte{byte(n%10) + '0'}, digits...)
		n /= 10
	}

	if negative {
		digits = append([]byte{'-'}, digits...)
	}

	return string(digits)
}
