package prometheus

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lk2023060901/xdooria/pkg/util/conc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Client Prometheus 客户端
type Client struct {
	config   *Config
	registry *prometheus.Registry

	// 指标存储
	counters   sync.Map // map[string]*prometheus.CounterVec
	gauges     sync.Map // map[string]*prometheus.GaugeVec
	histograms sync.Map // map[string]*prometheus.HistogramVec
	summaries  sync.Map // map[string]*prometheus.SummaryVec

	// HTTP 服务器
	httpServer *http.Server

	// 状态
	closed atomic.Bool
}

// New 创建 Prometheus 客户端
func New(cfg *Config) (*Client, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	c := &Client{
		config:   cfg,
		registry: prometheus.NewRegistry(),
	}

	// 注册默认采集器
	if cfg.EnableGoCollector {
		c.registry.MustRegister(collectors.NewGoCollector())
	}

	if cfg.EnableProcessCollector {
		c.registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	}

	// 启动 HTTP 服务器
	if cfg.HTTPServer.Enabled {
		if err := c.startHTTPServer(); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// Registry 获取底层 Registry（高级用户使用）
func (c *Client) Registry() *prometheus.Registry {
	return c.registry
}

// Handler 返回 HTTP Handler（用于集成到现有 HTTP 服务器）
func (c *Client) Handler() http.Handler {
	return promhttp.HandlerFor(
		c.registry,
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		},
	)
}

// Config 获取配置
func (c *Client) Config() *Config {
	return c.config
}

// startHTTPServer 启动独立的 HTTP 服务器
func (c *Client) startHTTPServer() error {
	mux := http.NewServeMux()
	mux.Handle(c.config.HTTPServer.Path, c.Handler())

	c.httpServer = &http.Server{
		Addr:         c.config.HTTPServer.Addr,
		Handler:      mux,
		ReadTimeout:  c.config.HTTPServer.Timeout,
		WriteTimeout: c.config.HTTPServer.Timeout,
	}

	conc.Go(func() (struct{}, error) {
		if err := c.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// 可以集成到框架的 Logger
			// log.Printf("prometheus: http server error: %v", err)
		}
		return struct{}{}, nil
	})

	return nil
}

// Close 关闭客户端
func (c *Client) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return ErrClientClosed
	}

	if c.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return c.httpServer.Shutdown(ctx)
	}

	return nil
}

// IsClosed 检查客户端是否已关闭
func (c *Client) IsClosed() bool {
	return c.closed.Load()
}
