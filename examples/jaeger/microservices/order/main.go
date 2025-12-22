package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lk2023060901/xdooria/examples/jaeger/microservices/common"
	"github.com/lk2023060901/xdooria/pkg/jaeger"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
)

type Order struct {
	OrderID   string    `json:"order_id"`
	UserID    string    `json:"user_id"`
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Amount    float64   `json:"amount"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateOrderRequest struct {
	UserID    string  `json:"user_id"`
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Amount    float64 `json:"amount"`
}

type CreateOrderResponse struct {
	Success bool    `json:"success"`
	Message string  `json:"message"`
	Order   *Order  `json:"order,omitempty"`
	Error   string  `json:"error,omitempty"`
}

var (
	tracer       *jaeger.Tracer
	inventoryURL string
	paymentURL   string
)

func main() {
	// 初始化 Tracer
	var err error
	tracer, err = common.InitTracer("order-service")
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer tracer.Close()

	// 获取下游服务地址
	inventoryURL = os.Getenv("INVENTORY_URL")
	if inventoryURL == "" {
		inventoryURL = "http://localhost:8082"
	}

	paymentURL = os.Getenv("PAYMENT_URL")
	if paymentURL == "" {
		paymentURL = "http://localhost:8083"
	}

	// 设置路由
	http.HandleFunc("/order/create", handleCreateOrder)
	http.HandleFunc("/health", handleHealth)

	// 启动服务器
	server := &http.Server{
		Addr:    ":8081",
		Handler: http.DefaultServeMux,
	}

	// 使用 conc.Go 启动 HTTP 服务器
	serverFuture := conc.Go(func() (struct{}, error) {
		log.Printf("Order service listening on :8081")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return struct{}{}, fmt.Errorf("server failed: %w", err)
		}
		return struct{}{}, nil
	})

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// 等待服务器 goroutine 完成
	if _, err := serverFuture.Await(); err != nil {
		log.Printf("Server error: %v", err)
	}

	log.Println("Server exited")
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求
	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 创建根 Span
	ctx, span := tracer.Start(r.Context(), "CreateOrder")
	defer span.End()

	// 添加订单信息到 Span
	span.SetAttributes(
		attribute.String("order.user_id", req.UserID),
		attribute.String("order.product_id", req.ProductID),
		attribute.Int("order.quantity", req.Quantity),
		attribute.Float64("order.amount", req.Amount),
	)

	// 执行订单创建流程
	order, err := createOrder(ctx, &req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		resp := CreateOrderResponse{
			Success: false,
			Message: "Failed to create order",
			Error:   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	span.SetStatus(codes.Ok, "Order created successfully")

	resp := CreateOrderResponse{
		Success: true,
		Message: "Order created successfully",
		Order:   order,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func createOrder(ctx context.Context, req *CreateOrderRequest) (*Order, error) {
	ctx, span := tracer.Start(ctx, "createOrder")
	defer span.End()

	// 1. 检查库存
	if err := checkInventory(ctx, req.ProductID, req.Quantity); err != nil {
		return nil, fmt.Errorf("inventory check failed: %w", err)
	}

	// 2. 处理支付
	if err := processPayment(ctx, req.UserID, req.Amount); err != nil {
		return nil, fmt.Errorf("payment processing failed: %w", err)
	}

	// 3. 扣减库存
	if err := deductInventory(ctx, req.ProductID, req.Quantity); err != nil {
		// 支付成功但扣减库存失败，需要退款（这里简化处理）
		return nil, fmt.Errorf("inventory deduction failed: %w", err)
	}

	// 4. 创建订单记录
	order := &Order{
		OrderID:   fmt.Sprintf("ORD-%d", time.Now().Unix()),
		UserID:    req.UserID,
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
		Amount:    req.Amount,
		Status:    "completed",
		CreatedAt: time.Now(),
	}

	span.AddEvent("Order created")

	return order, nil
}

func checkInventory(ctx context.Context, productID string, quantity int) error {
	ctx, span := tracer.Start(ctx, "checkInventory")
	defer span.End()

	span.SetAttributes(
		attribute.String("product_id", productID),
		attribute.Int("quantity", quantity),
	)

	url := fmt.Sprintf("%s/inventory/check?product_id=%s&quantity=%d", inventoryURL, productID, quantity)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	// 注入追踪上下文
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		span.RecordError(err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("inventory check failed with status: %d", resp.StatusCode)
		span.RecordError(err)
		return err
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if available, ok := result["available"].(bool); !ok || !available {
		err := fmt.Errorf("insufficient inventory")
		span.RecordError(err)
		return err
	}

	return nil
}

func deductInventory(ctx context.Context, productID string, quantity int) error {
	ctx, span := tracer.Start(ctx, "deductInventory")
	defer span.End()

	span.SetAttributes(
		attribute.String("product_id", productID),
		attribute.Int("quantity", quantity),
	)

	url := fmt.Sprintf("%s/inventory/deduct?product_id=%s&quantity=%d", inventoryURL, productID, quantity)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}

	// 注入追踪上下文
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		span.RecordError(err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("inventory deduction failed with status: %d", resp.StatusCode)
		span.RecordError(err)
		return err
	}

	return nil
}

func processPayment(ctx context.Context, userID string, amount float64) error {
	ctx, span := tracer.Start(ctx, "processPayment")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.Float64("amount", amount),
	)

	url := fmt.Sprintf("%s/payment/process?user_id=%s&amount=%.2f", paymentURL, userID, amount)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}

	// 注入追踪上下文
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		span.RecordError(err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("payment processing failed with status: %d", resp.StatusCode)
		span.RecordError(err)
		return err
	}

	return nil
}
