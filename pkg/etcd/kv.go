package etcd

import (
	"context"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/api/v3/mvccpb"
)

// KV 键值操作客户端
type KV struct {
	client *Client
	kv     clientv3.KV
}

// newKV 创建 KV 客户端
func newKV(client *Client) *KV {
	return &KV{
		client: client,
		kv:     clientv3.NewKV(client.rawClient()),
	}
}

// Get 获取键值
func (k *KV) Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*KeyValue, error) {
	resp, err := k.kv.Get(ctx, key, opts...)
	if err != nil {
		return nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, ErrKeyNotFound
	}

	return kvToKeyValue(resp.Kvs[0]), nil
}

// GetWithPrefix 获取前缀匹配的所有键值
func (k *KV) GetWithPrefix(ctx context.Context, prefix string, opts ...clientv3.OpOption) ([]*KeyValue, error) {
	opts = append(opts, clientv3.WithPrefix())
	resp, err := k.kv.Get(ctx, prefix, opts...)
	if err != nil {
		return nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, ErrKeyNotFound
	}

	result := make([]*KeyValue, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		result = append(result, kvToKeyValue(kv))
	}

	return result, nil
}

// Put 设置键值
func (k *KV) Put(ctx context.Context, key, value string, opts ...clientv3.OpOption) error {
	_, err := k.kv.Put(ctx, key, value, opts...)
	return err
}

// PutWithLease 设置键值并绑定租约
func (k *KV) PutWithLease(ctx context.Context, key, value string, leaseID LeaseID, opts ...clientv3.OpOption) error {
	opts = append(opts, clientv3.WithLease(clientv3.LeaseID(leaseID)))
	_, err := k.kv.Put(ctx, key, value, opts...)
	return err
}

// Delete 删除键
func (k *KV) Delete(ctx context.Context, key string, opts ...clientv3.OpOption) (int64, error) {
	resp, err := k.kv.Delete(ctx, key, opts...)
	if err != nil {
		return 0, err
	}
	return resp.Deleted, nil
}

// DeleteWithPrefix 删除前缀匹配的所有键
func (k *KV) DeleteWithPrefix(ctx context.Context, prefix string, opts ...clientv3.OpOption) (int64, error) {
	opts = append(opts, clientv3.WithPrefix())
	resp, err := k.kv.Delete(ctx, prefix, opts...)
	if err != nil {
		return 0, err
	}
	return resp.Deleted, nil
}

// PutIfNotExists 如果键不存在则设置（原子操作）
func (k *KV) PutIfNotExists(ctx context.Context, key, value string, opts ...clientv3.OpOption) (bool, error) {
	// 使用事务实现 CAS
	txn := k.kv.Txn(ctx)
	resp, err := txn.
		If(clientv3.Compare(clientv3.Version(key), "=", 0)). // 版本为 0 表示不存在
		Then(clientv3.OpPut(key, value, opts...)).
		Commit()

	if err != nil {
		return false, err
	}

	return resp.Succeeded, nil
}

// CompareAndSwap 比较并交换（CAS）
func (k *KV) CompareAndSwap(ctx context.Context, key, oldValue, newValue string, opts ...clientv3.OpOption) (bool, error) {
	txn := k.kv.Txn(ctx)
	resp, err := txn.
		If(clientv3.Compare(clientv3.Value(key), "=", oldValue)).
		Then(clientv3.OpPut(key, newValue, opts...)).
		Commit()

	if err != nil {
		return false, err
	}

	return resp.Succeeded, nil
}

// CompareAndDelete 比较并删除
func (k *KV) CompareAndDelete(ctx context.Context, key, value string) (bool, error) {
	txn := k.kv.Txn(ctx)
	resp, err := txn.
		If(clientv3.Compare(clientv3.Value(key), "=", value)).
		Then(clientv3.OpDelete(key)).
		Commit()

	if err != nil {
		return false, err
	}

	return resp.Succeeded, nil
}

// kvToKeyValue 转换 etcd KeyValue 到自定义类型
func kvToKeyValue(kv *mvccpb.KeyValue) *KeyValue {
	return &KeyValue{
		Key:            string(kv.Key),
		Value:          kv.Value,
		CreateRevision: kv.CreateRevision,
		ModRevision:    kv.ModRevision,
		Version:        kv.Version,
		Lease:          int64(kv.Lease),
	}
}
