package client

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNewPoolMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()

	metrics := NewPoolMetrics("test", "pool", "localhost:50051", reg)

	if metrics == nil {
		t.Fatal("Expected non-nil metrics")
	}

	// 验证指标已注册
	count, err := testutil.GatherAndCount(reg)
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// 应该有 9 个指标
	expectedCount := 9
	if count != expectedCount {
		t.Errorf("Expected %d metrics, got %d", expectedCount, count)
	}
}

func TestPoolMetrics_UpdatePoolStats(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewPoolMetrics("test", "pool", "localhost:50051", reg)

	// 更新指标
	metrics.UpdatePoolStats(10, 3, 7, 10)

	// 验证 capacity
	if value := testutil.ToFloat64(metrics.poolCapacity); value != 10 {
		t.Errorf("Expected poolCapacity 10, got %f", value)
	}

	// 验证 active
	if value := testutil.ToFloat64(metrics.activeConnections); value != 3 {
		t.Errorf("Expected activeConnections 3, got %f", value)
	}

	// 验证 idle
	if value := testutil.ToFloat64(metrics.idleConnections); value != 7 {
		t.Errorf("Expected idleConnections 7, got %f", value)
	}

	// 验证 total
	if value := testutil.ToFloat64(metrics.totalConnections); value != 10 {
		t.Errorf("Expected totalConnections 10, got %f", value)
	}
}

func TestPoolMetrics_RecordGetConnection(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewPoolMetrics("test", "pool", "localhost:50051", reg)

	// 记录成功获取连接
	metrics.RecordGetConnection(10*time.Millisecond, nil, false)

	// 验证 histogram 有数据
	count := testutil.CollectAndCount(metrics.getConnectionDuration)
	if count != 1 {
		t.Errorf("Expected 1 observation in histogram, got %d", count)
	}

	// 记录失败获取连接
	metrics.RecordGetConnection(50*time.Millisecond, ErrPoolClosed, false)

	// 验证错误计数
	if value := testutil.ToFloat64(metrics.getConnectionErrors); value != 1 {
		t.Errorf("Expected getConnectionErrors 1, got %f", value)
	}

	// 记录超时
	metrics.RecordGetConnection(100*time.Millisecond, ErrPoolExhausted, true)

	// 验证超时计数
	if value := testutil.ToFloat64(metrics.getConnectionTimeouts); value != 1 {
		t.Errorf("Expected getConnectionTimeouts 1, got %f", value)
	}
}

func TestPoolMetrics_RecordHealthCheckFailure(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewPoolMetrics("test", "pool", "localhost:50051", reg)

	// 记录健康检查失败
	metrics.RecordHealthCheckFailure()
	metrics.RecordHealthCheckFailure()

	// 验证计数
	if value := testutil.ToFloat64(metrics.healthCheckFailures); value != 2 {
		t.Errorf("Expected healthCheckFailures 2, got %f", value)
	}
}

func TestPoolMetrics_RecordConnectionRecreation(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewPoolMetrics("test", "pool", "localhost:50051", reg)

	// 记录连接重建
	metrics.RecordConnectionRecreation()
	metrics.RecordConnectionRecreation()
	metrics.RecordConnectionRecreation()

	// 验证计数
	if value := testutil.ToFloat64(metrics.connectionRecreations); value != 3 {
		t.Errorf("Expected connectionRecreations 3, got %f", value)
	}
}

func TestPoolMetrics_Unregister(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewPoolMetrics("test", "pool", "localhost:50051", reg)

	// 验证指标已注册
	count, _ := testutil.GatherAndCount(reg)
	if count == 0 {
		t.Error("Expected metrics to be registered")
	}

	// 取消注册
	metrics.Unregister(reg)

	// 验证指标已取消注册
	count, _ = testutil.GatherAndCount(reg)
	if count != 0 {
		t.Errorf("Expected 0 metrics after unregister, got %d", count)
	}
}

func TestPoolMetrics_DefaultRegisterer(t *testing.T) {
	// 测试使用默认 registerer（nil）
	// 注意：这会影响全局的 prometheus.DefaultRegisterer，所以需要谨慎

	metrics := NewPoolMetrics("test", "pool", "localhost:50051", nil)

	if metrics == nil {
		t.Fatal("Expected non-nil metrics with default registerer")
	}

	// 清理：从默认 registerer 取消注册
	metrics.Unregister(nil)
}

func BenchmarkPoolMetrics_UpdatePoolStats(b *testing.B) {
	reg := prometheus.NewRegistry()
	metrics := NewPoolMetrics("test", "pool", "localhost:50051", reg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.UpdatePoolStats(10, 5, 5, 10)
	}
}

func BenchmarkPoolMetrics_RecordGetConnection(b *testing.B) {
	reg := prometheus.NewRegistry()
	metrics := NewPoolMetrics("test", "pool", "localhost:50051", reg)

	duration := 10 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordGetConnection(duration, nil, false)
	}
}
