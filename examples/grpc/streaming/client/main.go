package main

import (
	"context"
	"io"
	"log"
	"time"

	pb "github.com/lk2023060901/xdooria/examples/grpc/proto/streaming"
	"github.com/lk2023060901/xdooria/pkg/grpc/client"
)

func main() {
	// 创建 gRPC Client
	cfg := &client.Config{
		Target:         "localhost:50052",
		DialTimeout:    5 * time.Second,
		RequestTimeout: 30 * time.Second, // 流式调用需要较长超时
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

	log.Println("Connected to streaming server")

	// 获取连接
	conn, err := cli.GetConn()
	if err != nil {
		log.Fatalf("Failed to get connection: %v", err)
	}

	// 创建 StreamService 客户端
	streamClient := pb.NewStreamServiceClient(conn)

	// 演示 1: Server Streaming
	log.Println("\n=== Server Streaming Demo ===")
	testServerStreaming(streamClient)

	time.Sleep(1 * time.Second)

	// 演示 2: Client Streaming
	log.Println("\n=== Client Streaming Demo ===")
	testClientStreaming(streamClient)

	time.Sleep(1 * time.Second)

	// 演示 3: Bidirectional Streaming
	log.Println("\n=== Bidirectional Streaming Demo ===")
	testBidirectionalStreaming(streamClient)
}

// testServerStreaming 测试服务端流式 RPC
func testServerStreaming(client pb.StreamServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := client.ListNumbers(ctx, &pb.ListRequest{Count: 5})
	if err != nil {
		log.Fatalf("ListNumbers failed: %v", err)
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			log.Println("Server finished sending")
			break
		}
		if err != nil {
			log.Fatalf("Recv failed: %v", err)
		}

		log.Printf("Received number: %d", resp.GetNumber())
	}
}

// testClientStreaming 测试客户端流式 RPC
func testClientStreaming(client pb.StreamServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := client.Sum(ctx)
	if err != nil {
		log.Fatalf("Sum failed: %v", err)
	}

	// 发送一系列数字
	numbers := []int32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	for _, num := range numbers {
		if err := stream.Send(&pb.NumberRequest{Number: num}); err != nil {
			log.Fatalf("Send failed: %v", err)
		}
		log.Printf("Sent number: %d", num)
		time.Sleep(200 * time.Millisecond)
	}

	// 关闭发送并接收结果
	resp, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("CloseAndRecv failed: %v", err)
	}

	log.Printf("Sum result: %d, Count: %d", resp.GetSum(), resp.GetCount())
}

// testBidirectionalStreaming 测试双向流式 RPC
func testBidirectionalStreaming(client pb.StreamServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := client.Chat(ctx)
	if err != nil {
		log.Fatalf("Chat failed: %v", err)
	}

	// 启动接收 goroutine
	done := make(chan bool)
	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				done <- true
				return
			}
			if err != nil {
				log.Printf("Recv error: %v", err)
				done <- true
				return
			}

			log.Printf("Received from %s: %s", resp.GetUser(), resp.GetMessage())
		}
	}()

	// 发送一系列消息
	messages := []string{"Hello", "How are you?", "Nice to meet you", "Goodbye"}
	for _, msg := range messages {
		if err := stream.Send(&pb.ChatMessage{
			User:      "Client",
			Message:   msg,
			Timestamp: time.Now().Unix(),
		}); err != nil {
			log.Fatalf("Send failed: %v", err)
		}
		log.Printf("Sent: %s", msg)
		time.Sleep(1 * time.Second)
	}

	// 关闭发送
	if err := stream.CloseSend(); err != nil {
		log.Fatalf("CloseSend failed: %v", err)
	}

	// 等待接收完成
	<-done
	log.Println("Chat completed")
}
