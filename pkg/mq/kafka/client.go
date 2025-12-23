package kafka

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/lk2023060901/xdooria/pkg/config"
	"github.com/lk2023060901/xdooria/pkg/logger"
)

// Client Kafka 客户端
type Client struct {
	config *Config
	logger logger.Logger

	// 生产者（按 topic 缓存）
	producers  map[string]*Producer
	producerMu sync.RWMutex

	// 消费者组
	consumers  map[string]*ConsumerGroup
	consumerMu sync.RWMutex

	// 中间件
	producerMiddlewares []ProducerMiddleware
	consumerMiddlewares []Middleware

	closed atomic.Bool
}

// New 创建 Kafka 客户端
func New(cfg *Config, opts ...ClientOption) (*Client, error) {
	newCfg, err := config.MergeConfig(DefaultConfig(), cfg)
	if err != nil {
		return nil, err
	}

	// 处理 Async 字段：如果用户显式传入 cfg 且 Async 为 false，则保留用户设置
	if cfg != nil && !cfg.Producer.Async {
		newCfg.Producer.Async = false
	}

	if err := newCfg.Validate(); err != nil {
		return nil, err
	}

	c := &Client{
		config:    newCfg,
		logger:    logger.Noop(),
		producers: make(map[string]*Producer),
		consumers: make(map[string]*ConsumerGroup),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// ClientOption 客户端选项
type ClientOption func(*Client)

// WithLogger 设置日志
func WithLogger(l logger.Logger) ClientOption {
	return func(c *Client) {
		if l != nil {
			c.logger = l
		}
	}
}

// WithProducerMiddleware 添加生产者中间件
func WithProducerMiddleware(mw ...ProducerMiddleware) ClientOption {
	return func(c *Client) {
		c.producerMiddlewares = append(c.producerMiddlewares, mw...)
	}
}

// WithConsumerMiddleware 添加消费者中间件
func WithConsumerMiddleware(mw ...Middleware) ClientOption {
	return func(c *Client) {
		c.consumerMiddlewares = append(c.consumerMiddlewares, mw...)
	}
}

// Producer 获取或创建指定 topic 的生产者
func (c *Client) Producer(topic string) *Producer {
	if c.closed.Load() {
		return nil
	}

	c.producerMu.RLock()
	p, exists := c.producers[topic]
	c.producerMu.RUnlock()

	if exists {
		return p
	}

	c.producerMu.Lock()
	defer c.producerMu.Unlock()

	// 双重检查
	if p, exists = c.producers[topic]; exists {
		return p
	}

	p = newProducer(c, topic)
	c.producers[topic] = p

	c.logger.Debug("producer created", "topic", topic)

	return p
}

// Subscribe 订阅主题（创建消费者组）
func (c *Client) Subscribe(topics []string, handler Handler, opts ...ConsumerOption) (*ConsumerGroup, error) {
	if c.closed.Load() {
		return nil, ErrClientClosed
	}

	if len(topics) == 0 {
		return nil, ErrNoTopics
	}

	if handler == nil {
		return nil, ErrNoHandler
	}

	cg, err := newConsumerGroup(c, topics, handler, opts...)
	if err != nil {
		return nil, err
	}

	key := cg.ID()
	c.consumerMu.Lock()
	c.consumers[key] = cg
	c.consumerMu.Unlock()

	c.logger.Info("consumer group created",
		"id", key,
		"topics", topics,
	)

	return cg, nil
}

// Publish 发布消息（便捷方法）
func (c *Client) Publish(ctx context.Context, topic string, msg *Message) error {
	if c.closed.Load() {
		return ErrClientClosed
	}

	msg.Topic = topic
	return c.Producer(topic).Publish(ctx, msg)
}

// PublishBatch 批量发布消息
func (c *Client) PublishBatch(ctx context.Context, topic string, msgs []*Message) error {
	if c.closed.Load() {
		return ErrClientClosed
	}

	for _, msg := range msgs {
		msg.Topic = topic
	}
	return c.Producer(topic).PublishBatch(ctx, msgs)
}

// PublishWithKey 发布带 Key 的消息
func (c *Client) PublishWithKey(ctx context.Context, topic, key string, value []byte) error {
	return c.Publish(ctx, topic, &Message{
		Key:   []byte(key),
		Value: value,
	})
}

// Close 关闭客户端
func (c *Client) Close() error {
	if c.closed.Swap(true) {
		return ErrClientClosed
	}

	var errs []error

	// 关闭所有消费者
	c.consumerMu.Lock()
	for id, cg := range c.consumers {
		if err := cg.Close(); err != nil {
			c.logger.Error("failed to close consumer group",
				"id", id,
				"error", err,
			)
			errs = append(errs, err)
		}
	}
	c.consumers = nil
	c.consumerMu.Unlock()

	// 关闭所有生产者
	c.producerMu.Lock()
	for topic, p := range c.producers {
		if err := p.Close(); err != nil {
			c.logger.Error("failed to close producer",
				"topic", topic,
				"error", err,
			)
			errs = append(errs, err)
		}
	}
	c.producers = nil
	c.producerMu.Unlock()

	c.logger.Info("kafka client closed")

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// IsClosed 是否已关闭
func (c *Client) IsClosed() bool {
	return c.closed.Load()
}

// HealthCheck 健康检查
func (c *Client) HealthCheck(ctx context.Context) error {
	if c.closed.Load() {
		return ErrClientClosed
	}

	dialer, err := newDialer(c.config)
	if err != nil {
		return err
	}

	conn, err := dialer.DialContext(ctx, "tcp", c.config.Brokers[0])
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Brokers()
	return err
}

// Config 获取配置（只读）
func (c *Client) Config() *Config {
	return c.config
}

// ListTopics 列出所有主题
func (c *Client) ListTopics(ctx context.Context) ([]string, error) {
	if c.closed.Load() {
		return nil, ErrClientClosed
	}

	dialer, err := newDialer(c.config)
	if err != nil {
		return nil, err
	}

	conn, err := dialer.DialContext(ctx, "tcp", c.config.Brokers[0])
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	partitions, err := conn.ReadPartitions()
	if err != nil {
		return nil, err
	}

	topicSet := make(map[string]struct{})
	for _, p := range partitions {
		topicSet[p.Topic] = struct{}{}
	}

	topics := make([]string, 0, len(topicSet))
	for topic := range topicSet {
		topics = append(topics, topic)
	}

	return topics, nil
}
