package main

import (
	"context"
	"flag"
	"time"

	"github.com/lk2023060901/xdooria/examples/grpc/proto/helloworld"
	"github.com/lk2023060901/xdooria/pkg/logger"
	registryetcd "github.com/lk2023060901/xdooria/pkg/registry/etcd"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var (
	name     = flag.String("name", "World", "name to greet")
	balancer = flag.String("balancer", "round_robin", "load balancer: round_robin or consistent_hash")
)

func main() {
	flag.Parse()

	// 注册 etcd resolver (只保存配置，不立即连接)
	if err := registryetcd.RegisterBuilder(&registryetcd.Config{
		Endpoints: []string{"127.0.0.1:2379"},
	}); err != nil {
		logger.Default().Fatal("failed to register resolver builder", zap.Error(err))
	}

	logger.Default().Info("etcd resolver registered successfully")

	// 根据参数选择负载均衡策略
	var conn *grpc.ClientConn
	var err error

	if *balancer == "consistent_hash" {
		logger.Default().Info("using consistent hash load balancer")
		conn, err = registryetcd.DialServiceWithConsistentHash(
			"etcd:///helloworld.Greeter",
			registryetcd.WithTimeout(10*time.Second),
		)
	} else {
		logger.Default().Info("using round robin load balancer")
		conn, err = registryetcd.DialServiceWithRoundRobin(
			"etcd:///helloworld.Greeter",
			registryetcd.WithTimeout(10*time.Second),
		)
	}

	if err != nil {
		logger.Default().Fatal("failed to dial service", zap.Error(err))
	}
	defer conn.Close()

	// 创建客户端
	client := helloworld.NewGreeterClient(conn)

	// 发送多次请求，观察负载均衡效果
	for i := 0; i < 10; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)

		// 如果使用一致性哈希，需要在 metadata 中添加 hash-key
		if *balancer == "consistent_hash" {
			ctx = registryetcd.WithHashKey(ctx, *name)
		}

		resp, err := client.SayHello(ctx, &helloworld.HelloRequest{
			Name: *name,
		})

		if err != nil {
			logger.Default().Error("failed to call SayHello",
				zap.Int("attempt", i+1),
				zap.Error(err),
			)
		} else {
			logger.Default().Info("received response",
				zap.Int("attempt", i+1),
				zap.String("message", resp.Message),
			)
		}

		cancel()
		time.Sleep(500 * time.Millisecond)
	}
}
