package etcd

import (
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// KeyValue 键值对
type KeyValue struct {
	Key            string
	Value          []byte
	CreateRevision int64
	ModRevision    int64
	Version        int64
	Lease          int64
}

// WatchEvent 监听事件
type WatchEvent struct {
	Type     WatchEventType
	Key      string
	Value    []byte
	Revision int64
}

// WatchEventType 事件类型
type WatchEventType int

const (
	EventTypePut    WatchEventType = 0
	EventTypeDelete WatchEventType = 1
)

// LeaseID 租约 ID
type LeaseID int64

// ElectionValue 选举值
type ElectionValue struct {
	Key      string
	Value    []byte
	Revision int64
	LeaseID  LeaseID
}

// CompareTarget 比较目标
type CompareTarget int

const (
	CompareVersion CompareTarget = iota
	CompareCreate
	CompareMod
	CompareValue
	CompareLease
)

// CompareResult 比较结果
type CompareResult int

const (
	CompareEqual CompareResult = iota
	CompareGreater
	CompareLess
	CompareNotEqual
)

// OpType 操作类型
type OpType int

const (
	OpTypeGet OpType = iota
	OpTypePut
	OpTypeDelete
	OpTypeTxn
)

// Op 操作
type Op struct {
	t     OpType
	key   string
	value []byte
	opts  []clientv3.OpOption
}

// TxnResponse 事务响应
type TxnResponse struct {
	Succeeded bool
	Responses []interface{}
}

// Config etcd 配置
type Config struct {
	// 连接配置
	Endpoints   []string      // etcd 节点地址
	DialTimeout time.Duration // 连接超时时间

	// 认证配置
	Username string // 用户名
	Password string // 密码

	// TLS 配置
	CertFile string // 客户端证书文件
	KeyFile  string // 客户端私钥文件
	CAFile   string // CA 证书文件

	// 其他配置
	AutoSyncInterval time.Duration // 自动同步间隔
	MaxCallSendMsgSize int         // 最大发送消息大小
	MaxCallRecvMsgSize int         // 最大接收消息大小

	// 重试配置
	EnableRetry   bool          // 启用重试
	MaxRetries    int           // 最大重试次数
	RetryInterval time.Duration // 重试间隔
}

// LockOption 锁选项
type LockOption func(*lockOptions)

type lockOptions struct {
	ttl     int64
	timeout time.Duration
}

// ElectionOption 选举选项
type ElectionOption func(*electionOptions)

type electionOptions struct {
	ttl int64
}

// WatchOption 监听选项
type WatchOption func(*watchOptions)

type watchOptions struct {
	revision  int64
	prevKV    bool
	prefix    bool
	createdNotify bool
}
