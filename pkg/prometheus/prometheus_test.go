package prometheus

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Namespace != "app" {
		t.Errorf("Expected Namespace=app, got %s", cfg.Namespace)
	}

	if !cfg.HTTPServer.Enabled {
		t.Error("Expected HTTPServer.Enabled=true")
	}

	if cfg.HTTPServer.Addr != ":9090" {
		t.Errorf("Expected Addr=:9090, got %s", cfg.HTTPServer.Addr)
	}

	if cfg.HTTPServer.Path != "/metrics" {
		t.Errorf("Expected Path=/metrics, got %s", cfg.HTTPServer.Path)
	}

	if !cfg.EnableGoCollector {
		t.Error("Expected EnableGoCollector=true")
	}

	if !cfg.EnableProcessCollector {
		t.Error("Expected EnableProcessCollector=true")
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "empty namespace",
			config: &Config{
				Namespace: "",
			},
			wantErr: true,
		},
		{
			name: "http server enabled without addr",
			config: &Config{
				Namespace: "test",
				HTTPServer: HTTPServerConfig{
					Enabled: true,
					Addr:    "",
				},
			},
			wantErr: true,
		},
		{
			name: "http server disabled",
			config: &Config{
				Namespace: "test",
				HTTPServer: HTTPServerConfig{
					Enabled: false,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	cfg := &Config{
		Namespace: "test",
		HTTPServer: HTTPServerConfig{
			Enabled: false, // 不启动 HTTP 服务器避免端口冲突
		},
		EnableGoCollector:      true,
		EnableProcessCollector: true,
	}

	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	if client.config.Namespace != "test" {
		t.Errorf("Expected Namespace=test, got %s", client.config.Namespace)
	}

	if client.registry == nil {
		t.Error("Expected registry to be initialized")
	}
}

func TestNewCounter(t *testing.T) {
	client, err := New(&Config{
		Namespace: "test",
		HTTPServer: HTTPServerConfig{
			Enabled: false,
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	// 创建 Counter
	counter, err := client.NewCounter("requests_total", "Total requests", []string{"method", "status"})
	if err != nil {
		t.Fatalf("NewCounter() error = %v", err)
	}

	if counter == nil {
		t.Fatal("Expected counter to be non-nil")
	}

	// 使用 Counter
	counter.WithLabelValues("GET", "200").Inc()
	counter.WithLabelValues("POST", "201").Add(5)

	// 重复创建应该返回错误
	_, err = client.NewCounter("requests_total", "Total requests", []string{"method", "status"})
	if err != ErrMetricExists {
		t.Errorf("Expected ErrMetricExists, got %v", err)
	}

	// 获取 Counter
	retrieved, ok := client.GetCounter("requests_total")
	if !ok {
		t.Error("Expected to find counter")
	}
	if retrieved != counter {
		t.Error("Expected retrieved counter to match original")
	}
}

func TestNewGauge(t *testing.T) {
	client, err := New(&Config{
		Namespace: "test",
		HTTPServer: HTTPServerConfig{
			Enabled: false,
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	gauge, err := client.NewGauge("active_connections", "Active connections", []string{"server"})
	if err != nil {
		t.Fatalf("NewGauge() error = %v", err)
	}

	if gauge == nil {
		t.Fatal("Expected gauge to be non-nil")
	}

	// 使用 Gauge
	gauge.WithLabelValues("server1").Set(100)
	gauge.WithLabelValues("server1").Inc()
	gauge.WithLabelValues("server1").Dec()
	gauge.WithLabelValues("server1").Add(10)
	gauge.WithLabelValues("server1").Sub(5)
}

func TestNewHistogram(t *testing.T) {
	client, err := New(&Config{
		Namespace: "test",
		HTTPServer: HTTPServerConfig{
			Enabled: false,
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	buckets := []float64{0.1, 0.5, 1, 5}
	histogram, err := client.NewHistogram("request_duration_seconds", "Request duration", []string{"method"}, buckets)
	if err != nil {
		t.Fatalf("NewHistogram() error = %v", err)
	}

	if histogram == nil {
		t.Fatal("Expected histogram to be non-nil")
	}

	// 使用 Histogram
	histogram.WithLabelValues("GET").Observe(0.25)
	histogram.WithLabelValues("POST").Observe(1.5)

	// 使用默认 buckets
	histogram2, err := client.NewHistogram("response_size_bytes", "Response size", []string{"endpoint"}, nil)
	if err != nil {
		t.Fatalf("NewHistogram() with default buckets error = %v", err)
	}
	if histogram2 == nil {
		t.Fatal("Expected histogram2 to be non-nil")
	}
}

func TestNewSummary(t *testing.T) {
	client, err := New(&Config{
		Namespace: "test",
		HTTPServer: HTTPServerConfig{
			Enabled: false,
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	objectives := map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001}
	summary, err := client.NewSummary("request_latency_seconds", "Request latency", []string{"api"}, objectives)
	if err != nil {
		t.Fatalf("NewSummary() error = %v", err)
	}

	if summary == nil {
		t.Fatal("Expected summary to be non-nil")
	}

	// 使用 Summary
	summary.WithLabelValues("users").Observe(0.123)
	summary.WithLabelValues("orders").Observe(0.456)

	// 使用默认 objectives
	summary2, err := client.NewSummary("query_duration_seconds", "Query duration", []string{"db"}, nil)
	if err != nil {
		t.Fatalf("NewSummary() with default objectives error = %v", err)
	}
	if summary2 == nil {
		t.Fatal("Expected summary2 to be non-nil")
	}
}

func TestMustMethods(t *testing.T) {
	client, err := New(&Config{
		Namespace: "test",
		HTTPServer: HTTPServerConfig{
			Enabled: false,
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	// MustNewCounter
	counter := client.MustNewCounter("must_counter", "Must counter", []string{"label"})
	if counter == nil {
		t.Fatal("Expected counter to be non-nil")
	}

	// MustNewGauge
	gauge := client.MustNewGauge("must_gauge", "Must gauge", []string{"label"})
	if gauge == nil {
		t.Fatal("Expected gauge to be non-nil")
	}

	// MustNewHistogram
	histogram := client.MustNewHistogram("must_histogram", "Must histogram", []string{"label"}, nil)
	if histogram == nil {
		t.Fatal("Expected histogram to be non-nil")
	}

	// MustNewSummary
	summary := client.MustNewSummary("must_summary", "Must summary", []string{"label"}, nil)
	if summary == nil {
		t.Fatal("Expected summary to be non-nil")
	}
}

func TestClientClose(t *testing.T) {
	client, err := New(&Config{
		Namespace: "test",
		HTTPServer: HTTPServerConfig{
			Enabled: false,
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if client.IsClosed() {
		t.Error("Expected client to be open")
	}

	err = client.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	if !client.IsClosed() {
		t.Error("Expected client to be closed")
	}

	// 重复关闭应该返回错误
	err = client.Close()
	if err != ErrClientClosed {
		t.Errorf("Expected ErrClientClosed, got %v", err)
	}

	// 关闭后创建指标应该失败
	_, err = client.NewCounter("after_close", "After close", nil)
	if err != ErrClientClosed {
		t.Errorf("Expected ErrClientClosed, got %v", err)
	}
}

func TestRegisterCollector(t *testing.T) {
	client, err := New(&Config{
		Namespace: "test",
		HTTPServer: HTTPServerConfig{
			Enabled: false,
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	// 创建自定义采集器
	customCollector := prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "test",
			Name:      "custom_metric",
			Help:      "Custom metric",
		},
		func() float64 {
			return float64(time.Now().Unix())
		},
	)

	err = client.RegisterCollector(customCollector)
	if err != nil {
		t.Fatalf("RegisterCollector() error = %v", err)
	}
}

func TestHandler(t *testing.T) {
	client, err := New(&Config{
		Namespace: "test",
		HTTPServer: HTTPServerConfig{
			Enabled: false,
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	handler := client.Handler()
	if handler == nil {
		t.Fatal("Expected handler to be non-nil")
	}
}

func TestRegistry(t *testing.T) {
	client, err := New(&Config{
		Namespace: "test",
		HTTPServer: HTTPServerConfig{
			Enabled: false,
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	registry := client.Registry()
	if registry == nil {
		t.Fatal("Expected registry to be non-nil")
	}
}
