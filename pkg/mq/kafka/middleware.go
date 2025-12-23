package kafka

import (
	"context"
	"time"

	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/otel"
)

// ===============================
// 消费者中间件
// ===============================

// LoggingMiddleware 消费者日志中间件
func LoggingMiddleware(log logger.Logger) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, msg *Message) error {
			start := time.Now()

			log.Debug("consuming message",
				"topic", msg.Topic,
				"partition", msg.Partition,
				"offset", msg.Offset,
				"key", string(msg.Key),
			)

			err := next(ctx, msg)

			duration := time.Since(start)
			if err != nil {
				log.Error("message consume failed",
					"topic", msg.Topic,
					"partition", msg.Partition,
					"offset", msg.Offset,
					"duration", duration,
					"error", err,
				)
			} else {
				log.Debug("message consumed",
					"topic", msg.Topic,
					"partition", msg.Partition,
					"offset", msg.Offset,
					"duration", duration,
				)
			}

			return err
		}
	}
}

// TracingMiddleware 消费者追踪中间件
func TracingMiddleware(tracerName string) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, msg *Message) error {
			tracer := otel.Tracer(tracerName)

			// 从消息头提取追踪上下文
			carrier := otel.MapCarrier{}
			for k, v := range msg.Headers {
				carrier[k] = v
			}
			ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)

			// 创建消费者 span
			ctx, span := tracer.Start(ctx, "kafka.consume",
				otel.WithSpanKind(otel.SpanKindConsumer),
				otel.WithAttributes(
					otel.String("messaging.system", "kafka"),
					otel.String("messaging.destination", msg.Topic),
					otel.Int("messaging.kafka.partition", msg.Partition),
					otel.Int64("messaging.kafka.offset", msg.Offset),
				),
			)
			defer span.End()

			if len(msg.Key) > 0 {
				span.SetAttributes(otel.String("messaging.kafka.message_key", string(msg.Key)))
			}

			err := next(ctx, msg)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(otel.CodeError, err.Error())
			}

			return err
		}
	}
}

// RecoveryMiddleware 恢复中间件（捕获 panic）
func RecoveryMiddleware(log logger.Logger) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, msg *Message) (err error) {
			defer func() {
				if r := recover(); r != nil {
					log.Error("consumer panic recovered",
						"topic", msg.Topic,
						"partition", msg.Partition,
						"offset", msg.Offset,
						"panic", r,
					)
					err = ErrConsumerPanic
				}
			}()
			return next(ctx, msg)
		}
	}
}

// RetryMiddleware 重试中间件
func RetryMiddleware(maxRetries int, backoff time.Duration) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, msg *Message) error {
			var lastErr error
			for i := 0; i <= maxRetries; i++ {
				if i > 0 {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(backoff * time.Duration(i)):
					}
				}

				lastErr = next(ctx, msg)
				if lastErr == nil {
					return nil
				}
			}
			return lastErr
		}
	}
}

// ===============================
// 生产者中间件
// ===============================

// ProducerLoggingMiddleware 生产者日志中间件
func ProducerLoggingMiddleware(log logger.Logger) ProducerMiddleware {
	return func(ctx context.Context, msg *Message, next func(context.Context, *Message) error) error {
		start := time.Now()

		log.Debug("publishing message",
			"topic", msg.Topic,
			"key", string(msg.Key),
		)

		err := next(ctx, msg)

		duration := time.Since(start)
		if err != nil {
			log.Error("message publish failed",
				"topic", msg.Topic,
				"key", string(msg.Key),
				"duration", duration,
				"error", err,
			)
		} else {
			log.Debug("message published",
				"topic", msg.Topic,
				"key", string(msg.Key),
				"duration", duration,
			)
		}

		return err
	}
}

// ProducerTracingMiddleware 生产者追踪中间件
func ProducerTracingMiddleware(tracerName string) ProducerMiddleware {
	return func(ctx context.Context, msg *Message, next func(context.Context, *Message) error) error {
		tracer := otel.Tracer(tracerName)

		// 创建生产者 span
		ctx, span := tracer.Start(ctx, "kafka.publish",
			otel.WithSpanKind(otel.SpanKindProducer),
			otel.WithAttributes(
				otel.String("messaging.system", "kafka"),
				otel.String("messaging.destination", msg.Topic),
			),
		)
		defer span.End()

		if len(msg.Key) > 0 {
			span.SetAttributes(otel.String("messaging.kafka.message_key", string(msg.Key)))
		}

		// 注入追踪上下文到消息头
		if msg.Headers == nil {
			msg.Headers = make(map[string]string)
		}
		carrier := otel.MapCarrier(msg.Headers)
		otel.GetTextMapPropagator().Inject(ctx, carrier)

		err := next(ctx, msg)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(otel.CodeError, err.Error())
		}

		return err
	}
}

// ProducerRecoveryMiddleware 生产者恢复中间件
func ProducerRecoveryMiddleware(log logger.Logger) ProducerMiddleware {
	return func(ctx context.Context, msg *Message, next func(context.Context, *Message) error) (err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("producer panic recovered",
					"topic", msg.Topic,
					"key", string(msg.Key),
					"panic", r,
				)
				err = ErrProducerPanic
			}
		}()
		return next(ctx, msg)
	}
}
