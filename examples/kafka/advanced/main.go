package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/lk2023060901/xdooria/pkg/jaeger"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/mq/kafka"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
)

// Order 订单示例结构体
type Order struct {
	ID        string    `json:"id"`
	Product   string    `json:"product"`
	Quantity  int       `json:"quantity"`
	Price     float64   `json:"price"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	// 获取证书目录路径
	// 优先使用环境变量，否则使用当前工作目录下的 certs 目录
	certsDir := os.Getenv("CERTS_DIR")
	if certsDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			cwd = "."
		}
		certsDir = filepath.Join(cwd, "certs")
	}

	// 检查是否使用 TLS 模式
	useTLS := os.Getenv("USE_TLS") == "true"
	useSASL := os.Getenv("USE_SASL") == "true"

	// 创建 logger
	log, err := logger.New(&logger.Config{
		Level:  "debug",
		Format: "text",
	})
	if err != nil {
		panic(err)
	}

	// 初始化 Jaeger Tracer（可选，用于分布式追踪）
	useTracing := os.Getenv("USE_TRACING") == "true"
	var tracer *jaeger.Tracer
	if useTracing {
		tracer, err = jaeger.New(&jaeger.Config{
			Enabled:     true,
			ServiceName: "kafka-advanced-example",
			Endpoint:    "localhost:4318", // OTLP HTTP endpoint
		})
		if err != nil {
			log.Warn("failed to create tracer, tracing disabled", "error", err)
		} else {
			defer tracer.Close()
			log.Info("tracing enabled", "service", "kafka-advanced-example", "endpoint", "localhost:4318")
		}
	}

	// 构建配置
	cfg := &kafka.Config{
		Brokers: []string{"localhost:9092"},
		Producer: kafka.ProducerConfig{
			Async:        false,
			RequiredAcks: -1, // 等待所有副本确认
			Compression:  "snappy",
		},
		Consumer: kafka.ConsumerConfig{
			GroupID:        "advanced-example-group",
			StartOffset:    -2, // 从最早位置开始
			Concurrency:    2,
			CommitInterval: 0, // 手动提交
		},
	}

	// SASL_SSL 配置（需要同时启用 TLS 和 SASL）
	if useTLS || useSASL {
		cfg.Brokers = []string{"localhost:9093"} // SASL_SSL 端口
		cfg.TLS = &kafka.TLSConfig{
			Enable:   true,
			CertFile: filepath.Join(certsDir, "client-cert.pem"),
			KeyFile:  filepath.Join(certsDir, "client-key.pem"),
			CAFile:   filepath.Join(certsDir, "ca-cert.pem"),
		}
		cfg.SASL = &kafka.SASLConfig{
			Mechanism: "PLAIN",
			Username:  "client",
			Password:  "client-secret",
		}
		log.Info("SASL_SSL enabled", "username", "client", "broker", "localhost:9093", "certs_dir", certsDir)
	}

	// 构建中间件列表
	producerMiddlewares := []kafka.ProducerMiddleware{
		kafka.ProducerLoggingMiddleware(log),
		kafka.ProducerRecoveryMiddleware(log),
	}
	consumerMiddlewares := []kafka.Middleware{
		kafka.LoggingMiddleware(log),
		kafka.RecoveryMiddleware(log),
		kafka.RetryMiddleware(2, 100*time.Millisecond),
	}

	// 如果启用了追踪，添加 TracingMiddleware
	if useTracing && tracer != nil {
		producerMiddlewares = append([]kafka.ProducerMiddleware{
			kafka.ProducerTracingMiddleware("kafka-producer"),
		}, producerMiddlewares...)
		consumerMiddlewares = append([]kafka.Middleware{
			kafka.TracingMiddleware("kafka-consumer"),
		}, consumerMiddlewares...)
	}

	// 创建 Kafka 客户端（带中间件）
	client, err := kafka.New(cfg,
		kafka.WithLogger(log),
		kafka.WithProducerMiddleware(producerMiddlewares...),
		kafka.WithConsumerMiddleware(consumerMiddlewares...),
	)
	if err != nil {
		log.Error("failed to create kafka client", "error", err)
		os.Exit(1)
	}
	defer client.Close()

	middlewareNames := []string{"logging", "recovery"}
	consumerMwNames := []string{"logging", "recovery", "retry(2, 100ms)"}
	if useTracing && tracer != nil {
		middlewareNames = append([]string{"tracing"}, middlewareNames...)
		consumerMwNames = append([]string{"tracing"}, consumerMwNames...)
	}
	log.Info("kafka client created with middlewares",
		"producer_middlewares", middlewareNames,
		"consumer_middlewares", consumerMwNames,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. 健康检查
	log.Info("=== HealthCheck ===")
	if err := client.HealthCheck(ctx); err != nil {
		log.Error("health check failed", "error", err)
		os.Exit(1)
	}
	log.Info("health check passed")

	// 2. 列出主题
	log.Info("=== ListTopics ===")
	topics, err := client.ListTopics(ctx)
	if err != nil {
		log.Warn("failed to list topics (may be empty)", "error", err)
	} else {
		log.Info("existing topics", "topics", topics)
	}

	// 3. 订阅主题（手动提交）
	topic := "advanced-example-topic"
	log.Info("=== Subscribe (manual commit) ===")

	consumer, err := client.Subscribe([]string{topic}, func(ctx context.Context, msg *kafka.Message) error {
		log.Info("received message",
			"topic", msg.Topic,
			"partition", msg.Partition,
			"offset", msg.Offset,
			"key", string(msg.Key),
			"headers", msg.Headers,
		)

		// 解析 JSON（如果有）
		if ct, ok := msg.Headers["content-type"]; ok && ct == "application/json" {
			var order Order
			if err := json.Unmarshal(msg.Value, &order); err == nil {
				log.Info("parsed order", "order_id", order.ID, "product", order.Product)
			}
		}

		return nil
	}, kafka.WithAutoCommit(false)) // 手动提交

	if err != nil {
		log.Error("failed to subscribe", "error", err)
		os.Exit(1)
	}

	// 启动消费者
	if err := consumer.Start(ctx); err != nil {
		log.Error("failed to start consumer", "error", err)
		os.Exit(1)
	}
	log.Info("consumer started", "topics", consumer.Topics(), "auto_commit", false)

	// 等待消费者准备就绪
	time.Sleep(2 * time.Second)

	// 4. Producer 功能演示
	log.Info("=== Producer Features ===")
	producer := client.Producer(topic)

	// 4.1 PublishWithKey
	log.Info("--- PublishWithKey ---")
	if err := producer.PublishWithKey(ctx, "user-1001", []byte("action: login")); err != nil {
		log.Error("PublishWithKey failed", "error", err)
	} else {
		log.Info("PublishWithKey succeeded", "key", "user-1001")
	}

	// 4.2 PublishJSON
	log.Info("--- PublishJSON ---")
	order := Order{
		ID:        "ORD-001",
		Product:   "Laptop",
		Quantity:  1,
		Price:     1299.99,
		CreatedAt: time.Now(),
	}
	orderJSON, _ := json.Marshal(order)
	if err := producer.PublishJSON(ctx, order.ID, orderJSON, map[string]string{
		"source": "advanced-example",
	}); err != nil {
		log.Error("PublishJSON failed", "error", err)
	} else {
		log.Info("PublishJSON succeeded", "order_id", order.ID)
	}

	// 4.3 PublishBatch
	log.Info("--- PublishBatch ---")
	batchMsgs := make([]*kafka.Message, 5)
	for i := 0; i < 5; i++ {
		batchMsgs[i] = &kafka.Message{
			Key:   []byte(fmt.Sprintf("batch-key-%d", i)),
			Value: []byte(fmt.Sprintf("batch-message-%d at %s", i, time.Now().Format(time.RFC3339))),
			Headers: map[string]string{
				"batch-index": fmt.Sprintf("%d", i),
			},
		}
	}
	if err := producer.PublishBatch(ctx, batchMsgs); err != nil {
		log.Error("PublishBatch failed", "error", err)
	} else {
		log.Info("PublishBatch succeeded", "count", len(batchMsgs))
	}

	// 4.4 Producer Stats
	log.Info("--- Producer Stats ---")
	pStats := producer.Stats()
	log.Info("producer stats",
		"produced", pStats.MessagesProduced,
		"succeeded", pStats.MessagesSucceeded,
		"failed", pStats.MessagesFailed,
		"last_message_time", pStats.LastMessageTime,
	)

	// 4.5 Producer.Topic
	log.Info("--- Producer.Topic ---")
	log.Info("producer topic", "topic", producer.Topic())

	// 4.6 Producer.IsClosed (should be false)
	log.Info("--- Producer.IsClosed ---")
	log.Info("producer is closed", "closed", producer.IsClosed())

	// 4.7 创建一个新 Producer 来测试 Close 方法
	log.Info("--- Producer.Close Test ---")
	closableProducer := client.Producer("close-test-topic")
	log.Info("closable producer created",
		"topic", closableProducer.Topic(),
		"is_closed", closableProducer.IsClosed(),
	)
	if err := closableProducer.Close(); err != nil {
		log.Error("failed to close producer", "error", err)
	} else {
		log.Info("producer closed successfully",
			"is_closed", closableProducer.IsClosed(),
		)
	}

	// 尝试向已关闭的 Producer 发送消息（应该失败）
	if err := closableProducer.Publish(ctx, &kafka.Message{
		Key:   []byte("test"),
		Value: []byte("should fail"),
	}); err != nil {
		log.Info("publish to closed producer failed as expected", "error", err)
	} else {
		log.Warn("publish to closed producer unexpectedly succeeded")
	}

	// 5. Client 便捷方法
	log.Info("=== Client Convenience Methods ===")

	// 5.1 Client.PublishWithKey
	if err := client.PublishWithKey(ctx, topic, "client-key-1", []byte("client message 1")); err != nil {
		log.Error("Client.PublishWithKey failed", "error", err)
	} else {
		log.Info("Client.PublishWithKey succeeded")
	}

	// 5.2 Client.PublishBatch
	clientBatch := []*kafka.Message{
		{Key: []byte("client-batch-1"), Value: []byte("client batch message 1")},
		{Key: []byte("client-batch-2"), Value: []byte("client batch message 2")},
	}
	if err := client.PublishBatch(ctx, topic, clientBatch); err != nil {
		log.Error("Client.PublishBatch failed", "error", err)
	} else {
		log.Info("Client.PublishBatch succeeded", "count", len(clientBatch))
	}

	// 等待消息被消费
	log.Info("Waiting for messages to be consumed...")
	time.Sleep(5 * time.Second)

	// 6. Consumer Stats
	log.Info("=== Consumer Stats ===")
	cStats := consumer.Stats()
	log.Info("consumer stats",
		"consumed", cStats.MessagesConsumed,
		"succeeded", cStats.MessagesSucceeded,
		"failed", cStats.MessagesFailed,
		"last_message_time", cStats.LastMessageTime,
	)

	// 6.1 Consumer.IsRunning
	log.Info("--- Consumer.IsRunning ---")
	log.Info("consumer is running", "running", consumer.IsRunning(), "state", consumer.State())

	// 6.2 Consumer.Close 测试（创建新消费者）
	log.Info("--- Consumer.Close Test ---")
	closableTopic := "close-consumer-test-topic"
	closableConsumer, err := client.Subscribe([]string{closableTopic}, func(ctx context.Context, msg *kafka.Message) error {
		log.Info("closable consumer received", "key", string(msg.Key))
		return nil
	}, kafka.WithAutoCommit(true), kafka.WithConcurrency(1))

	if err != nil {
		log.Error("failed to create closable consumer", "error", err)
	} else {
		if err := closableConsumer.Start(ctx); err != nil {
			log.Error("failed to start closable consumer", "error", err)
		} else {
			log.Info("closable consumer started",
				"running", closableConsumer.IsRunning(),
				"state", closableConsumer.State(),
			)

			// 使用 Close 而不是 Stop
			if err := closableConsumer.Close(); err != nil {
				log.Error("failed to close consumer", "error", err)
			} else {
				log.Info("consumer closed successfully",
					"running", closableConsumer.IsRunning(),
					"state", closableConsumer.State(),
				)
			}
		}
	}

	// 7. Consumer Lag
	log.Info("=== Consumer Lag ===")
	lag, err := consumer.Lag(ctx)
	if err != nil {
		log.Warn("failed to get lag", "error", err)
	} else {
		for tp, l := range lag {
			log.Info("lag", "topic", tp.Topic, "partition", tp.Partition, "lag", l)
		}
	}

	// 8. Handler 错误测试（消费者失败场景）
	log.Info("=== Handler Error Test ===")
	errorTopic := "error-handler-test-topic"
	errorCount := 0
	errorConsumer, err := client.Subscribe([]string{errorTopic}, func(ctx context.Context, msg *kafka.Message) error {
		errorCount++
		if errorCount <= 2 {
			log.Info("error handler: returning error",
				"key", string(msg.Key),
				"attempt", errorCount,
			)
			return fmt.Errorf("simulated error for message: %s", string(msg.Key))
		}
		log.Info("error handler: processing success",
			"key", string(msg.Key),
		)
		return nil
	}, kafka.WithAutoCommit(true), kafka.WithConcurrency(1))

	if err != nil {
		log.Error("failed to create error consumer", "error", err)
	} else {
		if err := errorConsumer.Start(ctx); err != nil {
			log.Error("failed to start error consumer", "error", err)
		} else {
			log.Info("error consumer started", "topic", errorTopic)

			// 等待消费者准备就绪
			time.Sleep(2 * time.Second)

			// 发送测试消息
			for i := 0; i < 3; i++ {
				if err := client.Publish(ctx, errorTopic, &kafka.Message{
					Key:   []byte(fmt.Sprintf("error-test-%d", i)),
					Value: []byte(fmt.Sprintf("error test message %d", i)),
				}); err != nil {
					log.Error("failed to publish to error topic", "error", err)
				} else {
					log.Info("published to error topic", "key", fmt.Sprintf("error-test-%d", i))
				}
			}

			time.Sleep(3 * time.Second)

			// 检查错误消费者统计
			errStats := errorConsumer.Stats()
			log.Info("error consumer stats",
				"consumed", errStats.MessagesConsumed,
				"succeeded", errStats.MessagesSucceeded,
				"failed", errStats.MessagesFailed,
			)

			// 停止错误消费者
			if err := errorConsumer.Stop(); err != nil {
				log.Error("failed to stop error consumer", "error", err)
			} else {
				log.Info("error consumer stopped")
			}
		}
	}

	// 9. 第二个消费者（自动提交模式）
	log.Info("=== Second Consumer (auto commit) ===")
	topic2 := "advanced-example-topic-2"
	consumer2, err := client.Subscribe([]string{topic2}, func(ctx context.Context, msg *kafka.Message) error {
		log.Info("consumer2 received",
			"topic", msg.Topic,
			"key", string(msg.Key),
		)
		return nil
	}, kafka.WithAutoCommit(true), kafka.WithConcurrency(1))

	if err != nil {
		log.Error("failed to create consumer2", "error", err)
	} else {
		if err := consumer2.Start(ctx); err != nil {
			log.Error("failed to start consumer2", "error", err)
		} else {
			log.Info("consumer2 started", "topic", topic2, "auto_commit", true)

			// 发送测试消息
			if err := client.Publish(ctx, topic2, &kafka.Message{
				Key:   []byte("auto-commit-test"),
				Value: []byte("testing auto commit"),
			}); err != nil {
				log.Error("failed to publish to topic2", "error", err)
			}

			time.Sleep(2 * time.Second)

			// 停止 consumer2
			log.Info("--- Stop consumer2 ---")
			if err := consumer2.Stop(); err != nil {
				log.Error("failed to stop consumer2", "error", err)
			} else {
				log.Info("consumer2 stopped")
			}

			// Consumer2 Stats
			c2Stats := consumer2.Stats()
			log.Info("consumer2 stats",
				"consumed", c2Stats.MessagesConsumed,
				"succeeded", c2Stats.MessagesSucceeded,
			)
		}
	}

	// 9. 后台持续发送消息
	log.Info("=== Background Producer ===")
	conc.Go(func() (struct{}, error) {
		for i := 0; i < 3; i++ {
			select {
			case <-ctx.Done():
				return struct{}{}, nil
			default:
			}

			msg := &kafka.Message{
				Key:   []byte(fmt.Sprintf("bg-key-%d", i)),
				Value: []byte(fmt.Sprintf("background message %d", i)),
			}
			if err := client.Publish(ctx, topic, msg); err != nil {
				log.Error("bg publish failed", "error", err)
			} else {
				log.Info("bg published", "key", string(msg.Key))
			}
			time.Sleep(time.Second)
		}
		return struct{}{}, nil
	})

	// 10. 再次检查列出主题
	time.Sleep(2 * time.Second)
	log.Info("=== ListTopics (after publishing) ===")
	topics, err = client.ListTopics(ctx)
	if err != nil {
		log.Warn("failed to list topics", "error", err)
	} else {
		log.Info("topics after publishing", "topics", topics)
	}

	// 11. 异步生产者模式测试（使用已存在的 topic 来避免自动创建延迟）
	log.Info("=== Async Producer Mode ===")
	asyncCfg := &kafka.Config{
		Brokers: cfg.Brokers,
		Producer: kafka.ProducerConfig{
			Async:        true,
			BatchSize:    10,
			BatchTimeout: 500 * time.Millisecond,
			RequiredAcks: -1,
			Compression:  "snappy",
		},
		Consumer: kafka.ConsumerConfig{
			GroupID: "async-producer-test-group",
		},
		TLS:  cfg.TLS,
		SASL: cfg.SASL,
	}

	asyncClient, err := kafka.New(asyncCfg, kafka.WithLogger(log))
	if err != nil {
		log.Error("failed to create async client", "error", err)
	} else {
		// 使用已存在的 topic 避免自动创建延迟问题
		asyncTopic := topic // "advanced-example-topic"
		asyncProducer := asyncClient.Producer(asyncTopic)

		// 异步模式发送多条消息
		start := time.Now()
		for i := 0; i < 10; i++ {
			if err := asyncProducer.Publish(ctx, &kafka.Message{
				Key:   []byte(fmt.Sprintf("async-key-%d", i)),
				Value: []byte(fmt.Sprintf("async message %d", i)),
			}); err != nil {
				log.Error("async publish failed", "error", err, "index", i)
			}
		}
		elapsed := time.Since(start)
		log.Info("async producer: 10 messages queued",
			"elapsed", elapsed,
			"async", true,
		)

		// 等待批量发送完成
		time.Sleep(time.Second)

		asyncStats := asyncProducer.Stats()
		log.Info("async producer stats",
			"produced", asyncStats.MessagesProduced,
			"succeeded", asyncStats.MessagesSucceeded,
			"failed", asyncStats.MessagesFailed,
		)

		if err := asyncProducer.Close(); err != nil {
			log.Error("failed to close async producer", "error", err)
		}
		if err := asyncClient.Close(); err != nil {
			log.Error("failed to close async client", "error", err)
		}
		log.Info("async producer test completed")
	}

	// 12. 不同压缩类型测试（使用已存在的 topic）
	log.Info("=== Compression Types Test ===")
	compressionTypes := []string{"gzip", "lz4", "zstd", "snappy", "none"}
	compressionTopic := topic // 使用已存在的 topic

	for _, compType := range compressionTypes {
		compCfg := &kafka.Config{
			Brokers: cfg.Brokers,
			Producer: kafka.ProducerConfig{
				Async:        false,
				RequiredAcks: -1,
				Compression:  compType,
			},
			Consumer: kafka.ConsumerConfig{
				GroupID: "compression-test-group",
			},
			TLS:  cfg.TLS,
			SASL: cfg.SASL,
		}

		compClient, err := kafka.New(compCfg, kafka.WithLogger(log))
		if err != nil {
			log.Error("failed to create compression client", "error", err, "compression", compType)
			continue
		}

		compProducer := compClient.Producer(compressionTopic)
		testData := []byte("This is a test message for compression testing. " +
			"Adding more content to make compression more effective. " +
			"Repeating data: AAAABBBBCCCCDDDDEEEEFFFFGGGGHHHHIIIIJJJJ")

		start := time.Now()
		if err := compProducer.Publish(ctx, &kafka.Message{
			Key:   []byte(fmt.Sprintf("comp-%s", compType)),
			Value: testData,
			Headers: map[string]string{
				"compression": compType,
			},
		}); err != nil {
			log.Error("compression publish failed", "error", err, "compression", compType)
		} else {
			elapsed := time.Since(start)
			log.Info("compression test succeeded",
				"compression", compType,
				"data_size", len(testData),
				"elapsed", elapsed,
			)
		}

		if err := compProducer.Close(); err != nil {
			log.Warn("failed to close compression producer", "error", err)
		}
		if err := compClient.Close(); err != nil {
			log.Warn("failed to close compression client", "error", err)
		}
	}

	// 13. InsecureSkipVerify TLS 测试
	log.Info("=== InsecureSkipVerify TLS Test ===")
	if useTLS || useSASL {
		insecureCfg := &kafka.Config{
			Brokers: []string{"localhost:9093"},
			Producer: kafka.ProducerConfig{
				Async:        false,
				RequiredAcks: -1,
				Compression:  "snappy",
			},
			Consumer: kafka.ConsumerConfig{
				GroupID: "insecure-tls-test-group",
			},
			TLS: &kafka.TLSConfig{
				Enable:             true,
				InsecureSkipVerify: true,
				// 不需要提供 CA 证书，因为跳过验证
			},
			SASL: &kafka.SASLConfig{
				Mechanism: "PLAIN",
				Username:  "client",
				Password:  "client-secret",
			},
		}

		insecureClient, err := kafka.New(insecureCfg, kafka.WithLogger(log))
		if err != nil {
			log.Error("failed to create insecure TLS client", "error", err)
		} else {
			// 健康检查
			if err := insecureClient.HealthCheck(ctx); err != nil {
				log.Error("insecure TLS health check failed", "error", err)
			} else {
				log.Info("insecure TLS health check passed",
					"insecure_skip_verify", true,
				)
			}

			// 发送测试消息（使用已存在的 topic）
			insecureTopic := topic
			if err := insecureClient.Publish(ctx, insecureTopic, &kafka.Message{
				Key:   []byte("insecure-test"),
				Value: []byte("message sent with InsecureSkipVerify=true"),
			}); err != nil {
				log.Error("insecure TLS publish failed", "error", err)
			} else {
				log.Info("insecure TLS publish succeeded",
					"topic", insecureTopic,
					"insecure_skip_verify", true,
				)
			}

			if err := insecureClient.Close(); err != nil {
				log.Warn("failed to close insecure client", "error", err)
			}
			log.Info("InsecureSkipVerify test completed")
		}
	} else {
		log.Info("skipping InsecureSkipVerify test (TLS not enabled)")
	}

	// 等待信号
	log.Info("=== Waiting for shutdown signal (Ctrl+C) ===")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Info("shutting down...")
	cancel()

	// 11. 停止消费者
	log.Info("=== Stop Consumer ===")
	if err := consumer.Stop(); err != nil {
		log.Error("failed to stop consumer", "error", err)
	} else {
		log.Info("consumer stopped", "state", consumer.State())
	}

	// 最终统计
	log.Info("=== Final Stats ===")
	finalPStats := producer.Stats()
	log.Info("final producer stats",
		"produced", finalPStats.MessagesProduced,
		"succeeded", finalPStats.MessagesSucceeded,
		"failed", finalPStats.MessagesFailed,
	)

	finalCStats := consumer.Stats()
	log.Info("final consumer stats",
		"consumed", finalCStats.MessagesConsumed,
		"succeeded", finalCStats.MessagesSucceeded,
		"failed", finalCStats.MessagesFailed,
	)

	log.Info("example completed")
}
