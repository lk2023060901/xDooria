package interceptor

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// MetricsConfig Metrics 拦截器配置
type MetricsConfig struct {
	// 是否启用（默认 true）
	Enabled bool

	// Prometheus 命名空间（默认 "grpc"）
	Namespace string

	// Prometheus 子系统（默认 "server" 或 "client"）
	Subsystem string

	// 是否记录消息大小（默认 false，有性能开销）
	EnableMessageSize bool
}

// DefaultServerMetricsConfig 默认 Server Metrics 配置
func DefaultServerMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		Enabled:           true,
		Namespace:         "grpc",
		Subsystem:         "server",
		EnableMessageSize: false,
	}
}

// DefaultClientMetricsConfig 默认 Client Metrics 配置
func DefaultClientMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		Enabled:           true,
		Namespace:         "grpc",
		Subsystem:         "client",
		EnableMessageSize: false,
	}
}

// ServerMetrics Server 端指标
type ServerMetrics struct {
	// 请求总数
	handledTotal *prometheus.CounterVec

	// 请求耗时
	handlingSeconds *prometheus.HistogramVec

	// 正在处理的请求数
	startedTotal *prometheus.GaugeVec

	// 消息大小（可选）
	msgReceivedBytes *prometheus.HistogramVec
	msgSentBytes     *prometheus.HistogramVec
}

// NewServerMetrics 创建 Server 端指标
func NewServerMetrics(cfg *MetricsConfig) *ServerMetrics {
	if cfg == nil {
		cfg = DefaultServerMetricsConfig()
	}

	m := &ServerMetrics{
		handledTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "handled_total",
				Help:      "Total number of RPCs completed on the server, regardless of success or failure.",
			},
			[]string{"grpc_method", "grpc_code"},
		),
		handlingSeconds: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "handling_seconds",
				Help:      "Histogram of response latency (seconds) of gRPC that had been application-level handled by the server.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"grpc_method"},
		),
		startedTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "started_total",
				Help:      "Total number of RPCs started on the server.",
			},
			[]string{"grpc_method"},
		),
	}

	if cfg.EnableMessageSize {
		m.msgReceivedBytes = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "msg_received_bytes",
				Help:      "Histogram of message sizes received (bytes).",
				Buckets:   prometheus.ExponentialBuckets(64, 4, 8),
			},
			[]string{"grpc_method"},
		)
		m.msgSentBytes = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "msg_sent_bytes",
				Help:      "Histogram of message sizes sent (bytes).",
				Buckets:   prometheus.ExponentialBuckets(64, 4, 8),
			},
			[]string{"grpc_method"},
		)
	}

	return m
}

// Register 注册指标到 Prometheus
func (m *ServerMetrics) Register(registry prometheus.Registerer) error {
	collectors := []prometheus.Collector{
		m.handledTotal,
		m.handlingSeconds,
		m.startedTotal,
	}

	if m.msgReceivedBytes != nil {
		collectors = append(collectors, m.msgReceivedBytes, m.msgSentBytes)
	}

	for _, collector := range collectors {
		if err := registry.Register(collector); err != nil {
			return err
		}
	}

	return nil
}

// ServerMetricsInterceptor Server 端 Metrics 拦截器（Unary）
func ServerMetricsInterceptor(metrics *ServerMetrics, cfg *MetricsConfig) grpc.UnaryServerInterceptor {
	if cfg == nil {
		cfg = DefaultServerMetricsConfig()
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if !cfg.Enabled {
			return handler(ctx, req)
		}

		method := info.FullMethod

		// 记录请求开始
		metrics.startedTotal.WithLabelValues(method).Inc()
		defer metrics.startedTotal.WithLabelValues(method).Dec()

		// 记录耗时
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start).Seconds()

		// 记录指标
		code := status.Code(err)
		metrics.handledTotal.WithLabelValues(method, code.String()).Inc()
		metrics.handlingSeconds.WithLabelValues(method).Observe(duration)

		return resp, err
	}
}

// StreamServerMetricsInterceptor Server 端 Metrics 拦截器（Stream）
func StreamServerMetricsInterceptor(metrics *ServerMetrics, cfg *MetricsConfig) grpc.StreamServerInterceptor {
	if cfg == nil {
		cfg = DefaultServerMetricsConfig()
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !cfg.Enabled {
			return handler(srv, ss)
		}

		method := info.FullMethod

		// 记录请求开始
		metrics.startedTotal.WithLabelValues(method).Inc()
		defer metrics.startedTotal.WithLabelValues(method).Dec()

		// 记录耗时
		start := time.Now()
		err := handler(srv, ss)
		duration := time.Since(start).Seconds()

		// 记录指标
		code := status.Code(err)
		metrics.handledTotal.WithLabelValues(method, code.String()).Inc()
		metrics.handlingSeconds.WithLabelValues(method).Observe(duration)

		return err
	}
}

// ClientMetrics Client 端指标
type ClientMetrics struct {
	// 请求总数
	handledTotal *prometheus.CounterVec

	// 请求耗时
	handlingSeconds *prometheus.HistogramVec

	// 正在处理的请求数
	startedTotal *prometheus.GaugeVec
}

// NewClientMetrics 创建 Client 端指标
func NewClientMetrics(cfg *MetricsConfig) *ClientMetrics {
	if cfg == nil {
		cfg = DefaultClientMetricsConfig()
	}

	return &ClientMetrics{
		handledTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "handled_total",
				Help:      "Total number of RPCs completed by the client, regardless of success or failure.",
			},
			[]string{"grpc_method", "grpc_code"},
		),
		handlingSeconds: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "handling_seconds",
				Help:      "Histogram of response latency (seconds) of the gRPC until it is finished by the application.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"grpc_method"},
		),
		startedTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "started_total",
				Help:      "Total number of RPCs started on the client.",
			},
			[]string{"grpc_method"},
		),
	}
}

// Register 注册指标到 Prometheus
func (m *ClientMetrics) Register(registry prometheus.Registerer) error {
	collectors := []prometheus.Collector{
		m.handledTotal,
		m.handlingSeconds,
		m.startedTotal,
	}

	for _, collector := range collectors {
		if err := registry.Register(collector); err != nil {
			return err
		}
	}

	return nil
}

// ClientMetricsInterceptor Client 端 Metrics 拦截器（Unary）
func ClientMetricsInterceptor(metrics *ClientMetrics, cfg *MetricsConfig) grpc.UnaryClientInterceptor {
	if cfg == nil {
		cfg = DefaultClientMetricsConfig()
	}

	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if !cfg.Enabled {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		// 记录请求开始
		metrics.startedTotal.WithLabelValues(method).Inc()
		defer metrics.startedTotal.WithLabelValues(method).Dec()

		// 记录耗时
		start := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		duration := time.Since(start).Seconds()

		// 记录指标
		code := status.Code(err)
		metrics.handledTotal.WithLabelValues(method, code.String()).Inc()
		metrics.handlingSeconds.WithLabelValues(method).Observe(duration)

		return err
	}
}
