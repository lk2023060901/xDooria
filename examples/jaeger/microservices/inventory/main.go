package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

// 简单的内存库存存储
type InventoryStore struct {
	mu    sync.RWMutex
	items map[string]int // productID -> quantity
}

func NewInventoryStore() *InventoryStore {
	return &InventoryStore{
		items: map[string]int{
			"PROD-001": 100,
			"PROD-002": 50,
			"PROD-003": 200,
		},
	}
}

func (s *InventoryStore) Check(productID string, quantity int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	available, exists := s.items[productID]
	return exists && available >= quantity
}

func (s *InventoryStore) Deduct(productID string, quantity int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	available, exists := s.items[productID]
	if !exists {
		return fmt.Errorf("product not found: %s", productID)
	}

	if available < quantity {
		return fmt.Errorf("insufficient inventory: available=%d, required=%d", available, quantity)
	}

	s.items[productID] = available - quantity
	return nil
}

func (s *InventoryStore) GetAll() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]int)
	for k, v := range s.items {
		result[k] = v
	}
	return result
}

var (
	tracer *jaeger.Tracer
	store  *InventoryStore
)

func main() {
	// 初始化 Tracer
	var err error
	tracer, err = common.InitTracer("inventory-service")
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer tracer.Close()

	// 初始化库存存储
	store = NewInventoryStore()

	// 设置路由
	http.HandleFunc("/inventory/check", handleCheckInventory)
	http.HandleFunc("/inventory/deduct", handleDeductInventory)
	http.HandleFunc("/inventory/list", handleListInventory)
	http.HandleFunc("/health", handleHealth)

	// 启动服务器
	server := &http.Server{
		Addr:    ":8082",
		Handler: http.DefaultServeMux,
	}

	// 使用 conc.Go 启动 HTTP 服务器
	serverFuture := conc.Go(func() (struct{}, error) {
		log.Printf("Inventory service listening on :8082")
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

func handleCheckInventory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 提取追踪上下文
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	ctx, span := tracer.Start(ctx, "CheckInventory")
	defer span.End()

	// 获取参数
	productID := r.URL.Query().Get("product_id")
	quantityStr := r.URL.Query().Get("quantity")

	if productID == "" || quantityStr == "" {
		http.Error(w, "Missing product_id or quantity", http.StatusBadRequest)
		return
	}

	quantity, err := strconv.Atoi(quantityStr)
	if err != nil {
		http.Error(w, "Invalid quantity", http.StatusBadRequest)
		return
	}

	span.SetAttributes(
		attribute.String("product_id", productID),
		attribute.Int("quantity", quantity),
	)

	// 模拟一些处理延迟
	time.Sleep(50 * time.Millisecond)

	available := store.Check(productID, quantity)

	span.SetAttributes(attribute.Bool("available", available))
	if available {
		span.SetStatus(codes.Ok, "Inventory available")
	} else {
		span.SetStatus(codes.Error, "Insufficient inventory")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"available":  available,
		"product_id": productID,
		"quantity":   quantity,
	})
}

func handleDeductInventory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 提取追踪上下文
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	ctx, span := tracer.Start(ctx, "DeductInventory")
	defer span.End()

	// 获取参数
	productID := r.URL.Query().Get("product_id")
	quantityStr := r.URL.Query().Get("quantity")

	if productID == "" || quantityStr == "" {
		http.Error(w, "Missing product_id or quantity", http.StatusBadRequest)
		return
	}

	quantity, err := strconv.Atoi(quantityStr)
	if err != nil {
		http.Error(w, "Invalid quantity", http.StatusBadRequest)
		return
	}

	span.SetAttributes(
		attribute.String("product_id", productID),
		attribute.Int("quantity", quantity),
	)

	// 模拟一些处理延迟
	time.Sleep(30 * time.Millisecond)

	if err := store.Deduct(productID, quantity); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	span.SetStatus(codes.Ok, "Inventory deducted successfully")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"product_id": productID,
		"quantity":   quantity,
		"message":    "Inventory deducted successfully",
	})
}

func handleListInventory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 提取追踪上下文
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	ctx, span := tracer.Start(ctx, "ListInventory")
	defer span.End()

	items := store.GetAll()

	span.SetAttributes(attribute.Int("total_products", len(items)))
	span.SetStatus(codes.Ok, "Inventory listed successfully")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items": items,
	})
}
