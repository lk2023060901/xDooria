package main

import (
	"context"
	"log"
	"time"

	pb "github.com/lk2023060901/xdooria/examples/grpc/proto/helloworld"
	"github.com/lk2023060901/xdooria/pkg/network/grpc/client"
)

func main() {
	// 创建 gRPC Client
	cfg := &client.Config{
		Target:         "localhost:50051",
		DialTimeout:    5 * time.Second,
		RequestTimeout: 3 * time.Second,
	}

	cli, err := client.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// 建立连接
	if err := cli.Dial(); err != nil {
		log.Fatalf("Failed to dial: %v", err)
	}
	defer cli.Close()

	log.Println("Connected to server")

	// 获取连接
	conn, err := cli.GetConn()
	if err != nil {
		log.Fatalf("Failed to get connection: %v", err)
	}

	// 创建 Greeter 客户端
	greeterClient := pb.NewGreeterClient(conn)

	// 调用 SayHello
	ctx := context.Background()
	resp, err := greeterClient.SayHello(ctx, &pb.HelloRequest{Name: "World"})
	if err != nil {
		log.Fatalf("Failed to call SayHello: %v", err)
	}

	log.Printf("Response: %s", resp.GetMessage())
}
