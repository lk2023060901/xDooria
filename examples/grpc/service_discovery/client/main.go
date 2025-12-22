package main

import (
	"context"
	"log"
	"time"

	pb "github.com/lk2023060901/xdooria/examples/grpc/proto/helloworld"
	etcdRegistry "github.com/lk2023060901/xdooria/pkg/registry/etcd"
)

func main() {
	// 注册 etcd resolver builder 到 gRPC
	registryCfg := &etcdRegistry.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	}

	if err := etcdRegistry.RegisterBuilder(registryCfg); err != nil {
		log.Fatalf("Failed to register resolver builder: %v", err)
	}

	// 使用服务发现建立连接（etcd:///service-name 格式）
	conn, err := etcdRegistry.DialServiceWithRoundRobin("etcd:///xdooria.greeter")
	if err != nil {
		log.Fatalf("Failed to dial service: %v", err)
	}
	defer conn.Close()

	log.Println("Connected to server via service discovery")

	// 创建 Greeter 客户端
	greeterClient := pb.NewGreeterClient(conn)

	// 调用 SayHello 多次，观察负载均衡
	for i := 0; i < 5; i++ {
		ctx := context.Background()
		resp, err := greeterClient.SayHello(ctx, &pb.HelloRequest{Name: "World"})
		if err != nil {
			log.Printf("Failed to call SayHello: %v", err)
			continue
		}

		log.Printf("Response %d: %s", i+1, resp.GetMessage())
		time.Sleep(500 * time.Millisecond)
	}
}
