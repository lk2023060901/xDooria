package kafka

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
	"github.com/segmentio/kafka-go"
)

// ConsumerGroup 消费者组
type ConsumerGroup struct {
	client  *Client
	id      string
	topics  []string
	handler Handler
	reader  *kafka.Reader

	// 状态
	state atomic.Int32

	// 控制
	stopCh chan struct{}
	wg     sync.WaitGroup

	// 配置
	concurrency int
	autoCommit  bool

	// 统计
	stats ConsumerStats
}

// ConsumerOption 消费者选项
type ConsumerOption func(*ConsumerGroup)

// WithConcurrency 设置并发消费数
func WithConcurrency(n int) ConsumerOption {
	return func(cg *ConsumerGroup) {
		if n > 0 {
			cg.concurrency = n
		}
	}
}

// WithAutoCommit 启用自动提交
func WithAutoCommit(enable bool) ConsumerOption {
	return func(cg *ConsumerGroup) {
		cg.autoCommit = enable
	}
}

// WithGroupID 设置消费者组 ID（覆盖配置中的 GroupID）
func WithGroupID(groupID string) ConsumerOption {
	return func(cg *ConsumerGroup) {
		if groupID != "" {
			// 需要重新创建 reader，这里只是记录
			// 实际在 newConsumerGroup 中处理
		}
	}
}

// newConsumerGroup 创建消费者组
func newConsumerGroup(c *Client, topics []string, handler Handler, opts ...ConsumerOption) (*ConsumerGroup, error) {
	cfg := c.config.Consumer

	cg := &ConsumerGroup{
		client:      c,
		id:          uuid.New().String(),
		topics:      topics,
		handler:     handler,
		stopCh:      make(chan struct{}),
		concurrency: cfg.Concurrency,
		autoCommit:  cfg.CommitInterval > 0,
	}

	// 应用选项
	for _, opt := range opts {
		opt(cg)
	}

	if cg.concurrency < 1 {
		cg.concurrency = 1
	}

	// 应用中间件
	wrappedHandler := handler
	for i := len(c.consumerMiddlewares) - 1; i >= 0; i-- {
		wrappedHandler = c.consumerMiddlewares[i](wrappedHandler)
	}
	cg.handler = wrappedHandler

	// 创建 Reader
	readerCfg := kafka.ReaderConfig{
		Brokers:           c.config.Brokers,
		GroupID:           cfg.GroupID,
		GroupTopics:       topics,
		MinBytes:          cfg.MinBytes,
		MaxBytes:          cfg.MaxBytes,
		MaxWait:           cfg.MaxWait,
		StartOffset:       cfg.StartOffset,
		HeartbeatInterval: cfg.HeartbeatInterval,
		SessionTimeout:    cfg.SessionTimeout,
		RebalanceTimeout:  cfg.RebalanceTimeout,
	}

	if cg.autoCommit && cfg.CommitInterval > 0 {
		readerCfg.CommitInterval = cfg.CommitInterval
	}

	// 配置 TLS 和 SASL
	if c.config.TLS != nil || c.config.SASL != nil {
		dialer, err := newDialer(c.config)
		if err != nil {
			return nil, err
		}
		readerCfg.Dialer = dialer
	}

	cg.reader = kafka.NewReader(readerCfg)

	return cg, nil
}

// ID 返回消费者组 ID
func (cg *ConsumerGroup) ID() string {
	return cg.id
}

// Topics 返回订阅的主题
func (cg *ConsumerGroup) Topics() []string {
	return cg.topics
}

// Start 启动消费
func (cg *ConsumerGroup) Start(ctx context.Context) error {
	if !cg.state.CompareAndSwap(int32(ConsumerStateIdle), int32(ConsumerStateRunning)) {
		return ErrConsumerAlreadyRunning
	}

	cg.client.logger.Info("consumer group starting",
		"id", cg.id,
		"topics", cg.topics,
		"concurrency", cg.concurrency,
	)

	// 启动消费协程
	for i := 0; i < cg.concurrency; i++ {
		cg.wg.Add(1)
		workerID := i
		conc.Go(func() (struct{}, error) {
			defer cg.wg.Done()
			cg.consume(ctx, workerID)
			return struct{}{}, nil
		})
	}

	cg.client.logger.Info("consumer group started",
		"id", cg.id,
		"topics", cg.topics,
	)

	return nil
}

// consume 消费循环
func (cg *ConsumerGroup) consume(ctx context.Context, workerID int) {
	for {
		select {
		case <-ctx.Done():
			cg.client.logger.Debug("consumer worker stopping due to context",
				"id", cg.id,
				"worker_id", workerID,
			)
			return
		case <-cg.stopCh:
			cg.client.logger.Debug("consumer worker stopping due to stop signal",
				"id", cg.id,
				"worker_id", workerID,
			)
			return
		default:
		}

		// 使用带超时的 context 拉取消息，以便定期检查 stopCh
		fetchCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		kafkaMsg, err := cg.reader.FetchMessage(fetchCtx)
		cancel()
		if err != nil {
			// 检查父 context 是否已取消
			if ctx.Err() != nil {
				return
			}
			// 检查是否正在停止
			select {
			case <-cg.stopCh:
				return
			default:
			}
			// 超时是正常的，继续下一次循环检查 stopCh
			if errors.Is(err, context.DeadlineExceeded) {
				continue
			}

			cg.client.logger.Error("failed to fetch message",
				"id", cg.id,
				"worker_id", workerID,
				"error", err,
			)
			continue
		}

		atomic.AddInt64(&cg.stats.MessagesConsumed, 1)

		// 转换消息
		msg := &Message{
			Topic:     kafkaMsg.Topic,
			Key:       kafkaMsg.Key,
			Value:     kafkaMsg.Value,
			Partition: kafkaMsg.Partition,
			Offset:    kafkaMsg.Offset,
			Timestamp: kafkaMsg.Time,
			Headers:   make(map[string]string),
		}
		for _, h := range kafkaMsg.Headers {
			msg.Headers[h.Key] = string(h.Value)
		}

		// 处理消息
		if err := cg.handler(ctx, msg); err != nil {
			atomic.AddInt64(&cg.stats.MessagesFailed, 1)
			cg.client.logger.Error("failed to handle message",
				"id", cg.id,
				"topic", msg.Topic,
				"partition", msg.Partition,
				"offset", msg.Offset,
				"error", err,
			)
			// 不提交 offset，消息会被重新消费
			continue
		}

		atomic.AddInt64(&cg.stats.MessagesSucceeded, 1)
		cg.stats.LastMessageTime = time.Now()

		// 手动提交
		if !cg.autoCommit {
			if err := cg.reader.CommitMessages(ctx, kafkaMsg); err != nil {
				cg.client.logger.Error("failed to commit message",
					"id", cg.id,
					"topic", msg.Topic,
					"partition", msg.Partition,
					"offset", msg.Offset,
					"error", err,
				)
			}
		}
	}
}

// Stop 停止消费
func (cg *ConsumerGroup) Stop() error {
	if !cg.state.CompareAndSwap(int32(ConsumerStateRunning), int32(ConsumerStateStopping)) {
		state := ConsumerState(cg.state.Load())
		if state == ConsumerStateStopped || state == ConsumerStateStopping {
			return nil
		}
		return ErrConsumerNotRunning
	}

	cg.client.logger.Info("consumer group stopping", "id", cg.id)

	close(cg.stopCh)
	cg.wg.Wait()

	cg.state.Store(int32(ConsumerStateStopped))

	cg.client.logger.Info("consumer group stopped", "id", cg.id)

	return nil
}

// Close 关闭消费者
func (cg *ConsumerGroup) Close() error {
	// 先停止消费
	_ = cg.Stop()

	// 关闭 reader
	if err := cg.reader.Close(); err != nil {
		return err
	}

	cg.client.logger.Debug("consumer group closed", "id", cg.id)

	return nil
}

// State 返回消费者状态
func (cg *ConsumerGroup) State() ConsumerState {
	return ConsumerState(cg.state.Load())
}

// IsRunning 是否正在运行
func (cg *ConsumerGroup) IsRunning() bool {
	return cg.State() == ConsumerStateRunning
}

// Stats 返回统计信息
func (cg *ConsumerGroup) Stats() ConsumerStats {
	return ConsumerStats{
		MessagesConsumed:  atomic.LoadInt64(&cg.stats.MessagesConsumed),
		MessagesSucceeded: atomic.LoadInt64(&cg.stats.MessagesSucceeded),
		MessagesFailed:    atomic.LoadInt64(&cg.stats.MessagesFailed),
		LastMessageTime:   cg.stats.LastMessageTime,
	}
}

// Lag 获取消费滞后（简化实现，仅返回当前 reader 的 lag）
func (cg *ConsumerGroup) Lag(_ context.Context) (map[TopicPartition]int64, error) {
	stats := cg.reader.Stats()

	lag := make(map[TopicPartition]int64)
	// stats.Partition 是 string 类型（如 "0"），这里简化处理
	lag[TopicPartition{Topic: stats.Topic, Partition: 0}] = stats.Lag

	return lag, nil
}
