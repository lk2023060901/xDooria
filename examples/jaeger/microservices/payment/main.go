package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
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

// 简单的支付记录存储
type PaymentStore struct {
	mu       sync.RWMutex
	payments map[string]*Payment
}

type Payment struct {
	PaymentID   string    `json:"payment_id"`
	UserID      string    `json:"user_id"`
	Amount      float64   `json:"amount"`
	Status      string    `json:"status"` // success, failed, pending
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

func NewPaymentStore() *PaymentStore {
	return &PaymentStore{
		payments: make(map[string]*Payment),
	}
}

func (s *PaymentStore) Create(userID string, amount float64) *Payment {
	s.mu.Lock()
	defer s.mu.Unlock()

	payment := &Payment{
		PaymentID: fmt.Sprintf("PAY-%d", time.Now().UnixNano()),
		UserID:    userID,
		Amount:    amount,
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	s.payments[payment.PaymentID] = payment
	return payment
}

func (s *PaymentStore) Complete(paymentID string, success bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	payment, exists := s.payments[paymentID]
	if !exists {
		return fmt.Errorf("payment not found: %s", paymentID)
	}

	if success {
		payment.Status = "success"
	} else {
		payment.Status = "failed"
	}
	payment.CompletedAt = time.Now()

	return nil
}

func (s *PaymentStore) GetAll() []*Payment {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Payment, 0, len(s.payments))
	for _, p := range s.payments {
		result = append(result, p)
	}
	return result
}

var (
	tracer *jaeger.Tracer
	store  *PaymentStore
)

func main() {
	// 初始化随机数生成器
	rand.Seed(time.Now().UnixNano())

	// 初始化 Tracer
	var err error
	tracer, err = common.InitTracer("payment-service")
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer tracer.Close()

	// 初始化支付存储
	store = NewPaymentStore()

	// 设置路由
	http.HandleFunc("/payment/process", handleProcessPayment)
	http.HandleFunc("/payment/list", handleListPayments)
	http.HandleFunc("/health", handleHealth)

	// 启动服务器
	server := &http.Server{
		Addr:    ":8083",
		Handler: http.DefaultServeMux,
	}

	// 使用 conc.Go 启动 HTTP 服务器
	serverFuture := conc.Go(func() (struct{}, error) {
		log.Printf("Payment service listening on :8083")
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

func handleProcessPayment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 提取追踪上下文
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	ctx, span := tracer.Start(ctx, "ProcessPayment")
	defer span.End()

	// 获取参数
	userID := r.URL.Query().Get("user_id")
	amountStr := r.URL.Query().Get("amount")

	if userID == "" || amountStr == "" {
		http.Error(w, "Missing user_id or amount", http.StatusBadRequest)
		return
	}

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		http.Error(w, "Invalid amount", http.StatusBadRequest)
		return
	}

	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.Float64("amount", amount),
	)

	// 创建支付记录
	payment := store.Create(userID, amount)
	span.AddEvent("Payment created")

	// 模拟支付处理流程
	success, err := processPayment(ctx, payment)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	if !success {
		err := fmt.Errorf("payment processing failed")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Payment failed")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Payment processing failed",
		})
		return
	}

	span.SetStatus(codes.Ok, "Payment processed successfully")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"payment_id": payment.PaymentID,
		"status":     payment.Status,
		"message":    "Payment processed successfully",
	})
}

func processPayment(ctx context.Context, payment *Payment) (bool, error) {
	ctx, span := tracer.Start(ctx, "processPayment")
	defer span.End()

	span.SetAttributes(
		attribute.String("payment_id", payment.PaymentID),
		attribute.Float64("amount", payment.Amount),
	)

	// 步骤 1: 验证账户
	if err := validateAccount(ctx, payment.UserID); err != nil {
		store.Complete(payment.PaymentID, false)
		return false, err
	}

	// 步骤 2: 检查余额
	if err := checkBalance(ctx, payment.UserID, payment.Amount); err != nil {
		store.Complete(payment.PaymentID, false)
		return false, err
	}

	// 步骤 3: 扣款
	if err := deductBalance(ctx, payment.UserID, payment.Amount); err != nil {
		store.Complete(payment.PaymentID, false)
		return false, err
	}

	// 步骤 4: 完成支付
	store.Complete(payment.PaymentID, true)
	span.AddEvent("Payment completed successfully")

	return true, nil
}

func validateAccount(ctx context.Context, userID string) error {
	ctx, span := tracer.Start(ctx, "validateAccount")
	defer span.End()

	span.SetAttributes(attribute.String("user_id", userID))

	// 模拟账户验证延迟
	time.Sleep(20 * time.Millisecond)

	// 简单验证：userID 不能为空
	if userID == "" {
		err := fmt.Errorf("invalid user_id")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Account validation failed")
		return err
	}

	span.SetStatus(codes.Ok, "Account validated")
	return nil
}

func checkBalance(ctx context.Context, userID string, amount float64) error {
	ctx, span := tracer.Start(ctx, "checkBalance")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.Float64("amount", amount),
	)

	// 模拟余额检查延迟
	time.Sleep(30 * time.Millisecond)

	// 模拟 5% 的概率余额不足
	if rand.Float64() < 0.05 {
		err := fmt.Errorf("insufficient balance")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Insufficient balance")
		return err
	}

	span.SetStatus(codes.Ok, "Balance sufficient")
	return nil
}

func deductBalance(ctx context.Context, userID string, amount float64) error {
	ctx, span := tracer.Start(ctx, "deductBalance")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.Float64("amount", amount),
	)

	// 模拟扣款延迟
	time.Sleep(40 * time.Millisecond)

	// 模拟 3% 的概率扣款失败
	if rand.Float64() < 0.03 {
		err := fmt.Errorf("deduction failed due to system error")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Deduction failed")
		return err
	}

	span.SetStatus(codes.Ok, "Balance deducted successfully")
	return nil
}

func handleListPayments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 提取追踪上下文
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	ctx, span := tracer.Start(ctx, "ListPayments")
	defer span.End()

	payments := store.GetAll()

	span.SetAttributes(attribute.Int("total_payments", len(payments)))
	span.SetStatus(codes.Ok, "Payments listed successfully")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"payments": payments,
	})
}
