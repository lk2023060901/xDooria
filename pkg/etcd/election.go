package etcd

import (
	"context"

	"go.etcd.io/etcd/client/v3/concurrency"
)

// Election 选举客户端
type Election struct {
	client *Client
}

// newElection 创建 Election 客户端
func newElection(client *Client) *Election {
	return &Election{
		client: client,
	}
}

// Elector 选举器
type Elector struct {
	session  *concurrency.Session
	election *concurrency.Election
	prefix   string
}

// NewElector 创建选举器
func (e *Election) NewElector(prefix string, opts ...ElectionOption) (*Elector, error) {
	options := &electionOptions{
		ttl: 60, // 默认 60 秒
	}

	for _, opt := range opts {
		opt(options)
	}

	// 创建 session
	session, err := concurrency.NewSession(
		e.client.rawClient(),
		concurrency.WithTTL(int(options.ttl)),
	)
	if err != nil {
		return nil, err
	}

	// 创建 election
	election := concurrency.NewElection(session, prefix)

	return &Elector{
		session:  session,
		election: election,
		prefix:   prefix,
	}, nil
}

// Campaign 参与竞选
func (elector *Elector) Campaign(ctx context.Context, value string) error {
	return elector.election.Campaign(ctx, value)
}

// Resign 退出竞选
func (elector *Elector) Resign(ctx context.Context) error {
	return elector.election.Resign(ctx)
}

// Leader 获取当前 Leader
func (elector *Elector) Leader(ctx context.Context) (*ElectionValue, error) {
	resp, err := elector.election.Leader(ctx)
	if err != nil {
		return nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, ErrNotLeader
	}

	kv := resp.Kvs[0]
	return &ElectionValue{
		Key:      string(kv.Key),
		Value:    kv.Value,
		Revision: kv.ModRevision,
		LeaseID:  LeaseID(kv.Lease),
	}, nil
}

// Observe 观察 Leader 变化
func (elector *Elector) Observe(ctx context.Context, handler func(*ElectionValue)) error {
	observeCh := elector.election.Observe(ctx)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return

			case resp, ok := <-observeCh:
				if !ok {
					return
				}

				if len(resp.Kvs) == 0 {
					continue
				}

				kv := resp.Kvs[0]
				event := &ElectionValue{
					Key:      string(kv.Key),
					Value:    kv.Value,
					Revision: kv.ModRevision,
					LeaseID:  LeaseID(kv.Lease),
				}

				handler(event)
			}
		}
	}()

	return nil
}

// IsLeader 判断是否是 Leader
func (elector *Elector) IsLeader(ctx context.Context) (bool, error) {
	leader, err := elector.Leader(ctx)
	if err != nil {
		if err == ErrNotLeader {
			return false, nil
		}
		return false, err
	}

	// 比较 Revision
	key := elector.election.Key()
	if key == leader.Key {
		return true, nil
	}

	return false, nil
}

// Key 获取当前选举键
func (elector *Elector) Key() string {
	return elector.election.Key()
}

// Close 关闭选举器
func (elector *Elector) Close() error {
	return elector.session.Close()
}

// WithElectionDo 在 Leader 身份下执行函数
func (e *Election) WithElectionDo(ctx context.Context, prefix, value string, fn func() error, opts ...ElectionOption) error {
	elector, err := e.NewElector(prefix, opts...)
	if err != nil {
		return err
	}
	defer elector.Close()

	// 参与竞选
	if err := elector.Campaign(ctx, value); err != nil {
		return err
	}
	defer elector.Resign(context.Background())

	// 执行函数
	return fn()
}

// --- Election Options ---

// WithElectionTTL 设置选举 TTL
func WithElectionTTL(ttl int64) ElectionOption {
	return func(o *electionOptions) {
		o.ttl = ttl
	}
}
