package main

import (
	"context"
	"log"
	"time"

	pb "github.com/lk2023060901/xdooria/examples/grpc/proto/helloworld"
	"github.com/lk2023060901/xdooria/pkg/etcd"
	"github.com/lk2023060901/xdooria/pkg/grpc/client"
	etcdRegistry "github.com/lk2023060901/xdooria/pkg/registry/etcd"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
)

func main() {
	// 创建 etcd 客户端
	etcdCfg := &etcd.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	}

	etcdClient, err := etcd.New(etcdCfg)
	if err != nil {
		log.Fatalf("Failed to create etcd client: %v", err)
	}
	defer etcdClient.Close()

	// 创建服务解析器
	resolverCfg := &etcdRegistry.Config{}
	resolver, err := etcdRegistry.NewResolver(etcdClient, resolverCfg)
	if err != nil {
		log.Fatalf("Failed to create resolver: %v", err)
	}

	// 注册 etcd resolver 到 gRPC
	etcdRegistry.RegisterResolver(resolver)

	// 创建 gRPC Client
	cfg := &client.Config{
		Target:         "etcd:///xdooria.greeter", // 使用 etcd:/// scheme
		DialTimeout:    5 * time.Second,
		RequestTimeout: 3 * time.Second,
		LoadBalancer:   roundrobin.Name, // 使用轮询负载均衡
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

	log.Println("Connected to server via service discovery")

	// 获取连接
	conn, err := cli.GetConn()
	if err != nil {
		log.Fatalf("Failed to get connection: %v", err)
	}

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
