package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/mq/kafka"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
)

func main() {
	// 创建 logger
	log, err := logger.New(&logger.Config{
		Level:  "debug",
		Format: "text",
	})
	if err != nil {
		panic(err)
	}

	// 创建 Kafka 客户端
	client, err := kafka.New(&kafka.Config{
		Brokers: []string{"localhost:9092"},
		Producer: kafka.ProducerConfig{
			Async:        false,
			RequiredAcks: -1, // 等待所有副本确认
		},
		Consumer: kafka.ConsumerConfig{
			GroupID:     "example-group",
			StartOffset: -2, // 从最早位置开始
			Concurrency: 2,
		},
	}, kafka.WithLogger(log))
	if err != nil {
		log.Error("failed to create kafka client", "error", err)
		os.Exit(1)
	}
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 订阅主题
	topic := "example-topic"
	consumer, err := client.Subscribe([]string{topic}, func(ctx context.Context, msg *kafka.Message) error {
		log.Info("received message",
			"topic", msg.Topic,
			"partition", msg.Partition,
			"offset", msg.Offset,
			"key", string(msg.Key),
			"value", string(msg.Value),
		)
		return nil
	})
	if err != nil {
		log.Error("failed to subscribe", "error", err)
		os.Exit(1)
	}

	// 启动消费者
	if err := consumer.Start(ctx); err != nil {
		log.Error("failed to start consumer", "error", err)
		os.Exit(1)
	}
	log.Info("consumer started", "topics", consumer.Topics())

	// 发布消息
	conc.Go(func() (struct{}, error) {
		producer := client.Producer(topic)
		for i := 0; i < 10; i++ {
			msg := &kafka.Message{
				Key:   []byte(fmt.Sprintf("key-%d", i)),
				Value: []byte(fmt.Sprintf("message-%d at %s", i, time.Now().Format(time.RFC3339))),
			}

			if err := producer.Publish(ctx, msg); err != nil {
				log.Error("failed to publish", "error", err)
				continue
			}
			log.Info("published message", "key", string(msg.Key))

			time.Sleep(time.Second)
		}
		return struct{}{}, nil
	})

	// 等待退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Info("shutting down...")
	cancel()

	// 打印统计
	stats := consumer.Stats()
	log.Info("consumer stats",
		"consumed", stats.MessagesConsumed,
		"succeeded", stats.MessagesSucceeded,
		"failed", stats.MessagesFailed,
	)
}
