package kafka

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/segmentio/kafka-go"
)

// Producer Kafka 生产者
type Producer struct {
	client *Client
	topic  string
	writer *kafka.Writer

	// 统计
	stats ProducerStats

	closed atomic.Bool
}

// newProducer 创建生产者
func newProducer(c *Client, topic string) *Producer {
	cfg := c.config.Producer

	writer := &kafka.Writer{
		Addr:                   kafka.TCP(c.config.Brokers...),
		Topic:                  topic,
		Balancer:               &kafka.LeastBytes{},
		BatchSize:              cfg.BatchSize,
		BatchTimeout:           cfg.BatchTimeout,
		MaxAttempts:            cfg.MaxRetries + 1,
		WriteTimeout:           cfg.WriteTimeout,
		ReadTimeout:            cfg.ReadTimeout,
		RequiredAcks:           kafka.RequiredAcks(cfg.RequiredAcks),
		Async:                  cfg.Async,
		Compression:            parseCompression(cfg.Compression),
		AllowAutoTopicCreation: true,
	}

	// 配置 TLS 和 SASL
	if c.config.TLS != nil || c.config.SASL != nil {
		transport, err := newTransport(c.config)
		if err == nil {
			writer.Transport = transport
		}
	}

	return &Producer{
		client: c,
		topic:  topic,
		writer: writer,
	}
}

// Publish 发布单条消息
func (p *Producer) Publish(ctx context.Context, msg *Message) error {
	if p.closed.Load() {
		return ErrProducerClosed
	}

	atomic.AddInt64(&p.stats.MessagesProduced, 1)

	// 构建中间件链
	publish := func(ctx context.Context, msg *Message) error {
		return p.doPublish(ctx, msg)
	}

	for i := len(p.client.producerMiddlewares) - 1; i >= 0; i-- {
		mw := p.client.producerMiddlewares[i]
		next := publish
		publish = func(ctx context.Context, msg *Message) error {
			return mw(ctx, msg, next)
		}
	}

	err := publish(ctx, msg)
	if err != nil {
		atomic.AddInt64(&p.stats.MessagesFailed, 1)
	} else {
		atomic.AddInt64(&p.stats.MessagesSucceeded, 1)
		p.stats.LastMessageTime = time.Now()
	}

	return err
}

// doPublish 实际发布
func (p *Producer) doPublish(ctx context.Context, msg *Message) error {
	kafkaMsg := kafka.Message{
		Key:   msg.Key,
		Value: msg.Value,
	}

	// 设置 headers
	if len(msg.Headers) > 0 {
		headers := make([]kafka.Header, 0, len(msg.Headers))
		for k, v := range msg.Headers {
			headers = append(headers, kafka.Header{Key: k, Value: []byte(v)})
		}
		kafkaMsg.Headers = headers
	}

	return p.writer.WriteMessages(ctx, kafkaMsg)
}

// PublishBatch 批量发布消息
func (p *Producer) PublishBatch(ctx context.Context, msgs []*Message) error {
	if p.closed.Load() {
		return ErrProducerClosed
	}

	if len(msgs) == 0 {
		return nil
	}

	atomic.AddInt64(&p.stats.MessagesProduced, int64(len(msgs)))

	kafkaMsgs := make([]kafka.Message, len(msgs))
	for i, msg := range msgs {
		kafkaMsgs[i] = kafka.Message{
			Key:   msg.Key,
			Value: msg.Value,
		}

		if len(msg.Headers) > 0 {
			headers := make([]kafka.Header, 0, len(msg.Headers))
			for k, v := range msg.Headers {
				headers = append(headers, kafka.Header{Key: k, Value: []byte(v)})
			}
			kafkaMsgs[i].Headers = headers
		}
	}

	err := p.writer.WriteMessages(ctx, kafkaMsgs...)
	if err != nil {
		atomic.AddInt64(&p.stats.MessagesFailed, int64(len(msgs)))
	} else {
		atomic.AddInt64(&p.stats.MessagesSucceeded, int64(len(msgs)))
		p.stats.LastMessageTime = time.Now()
	}

	return err
}

// PublishWithKey 发布带 Key 的消息
func (p *Producer) PublishWithKey(ctx context.Context, key string, value []byte) error {
	return p.Publish(ctx, &Message{
		Key:   []byte(key),
		Value: value,
	})
}

// PublishJSON 发布 JSON 消息
func (p *Producer) PublishJSON(ctx context.Context, key string, value []byte, headers map[string]string) error {
	if headers == nil {
		headers = make(map[string]string)
	}
	headers["content-type"] = "application/json"

	return p.Publish(ctx, &Message{
		Key:     []byte(key),
		Value:   value,
		Headers: headers,
	})
}

// Topic 返回 topic 名称
func (p *Producer) Topic() string {
	return p.topic
}

// Stats 返回统计信息
func (p *Producer) Stats() ProducerStats {
	return ProducerStats{
		MessagesProduced:  atomic.LoadInt64(&p.stats.MessagesProduced),
		MessagesSucceeded: atomic.LoadInt64(&p.stats.MessagesSucceeded),
		MessagesFailed:    atomic.LoadInt64(&p.stats.MessagesFailed),
		LastMessageTime:   p.stats.LastMessageTime,
	}
}

// Close 关闭生产者
func (p *Producer) Close() error {
	if p.closed.Swap(true) {
		return nil
	}

	p.client.logger.Debug("producer closing", "topic", p.topic)

	return p.writer.Close()
}

// IsClosed 是否已关闭
func (p *Producer) IsClosed() bool {
	return p.closed.Load()
}

// parseCompression 解析压缩算法
func parseCompression(s string) kafka.Compression {
	switch s {
	case "gzip":
		return kafka.Gzip
	case "snappy":
		return kafka.Snappy
	case "lz4":
		return kafka.Lz4
	case "zstd":
		return kafka.Zstd
	default:
		return 0 // none
	}
}
