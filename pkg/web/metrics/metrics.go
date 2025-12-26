package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// HttpRequestsTotal HTTP 请求总数
	HttpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "http",
			Subsystem: "server",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests.",
		},
		[]string{"path", "method", "status"},
	)

	// HttpRequestDuration HTTP 请求耗时
	HttpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "http",
			Subsystem: "server",
			Name:      "request_duration_seconds",
			Help:      "HTTP request latency in seconds.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"path", "method"},
	)
)

// InitMetrics 注册 Web 指标到指定的注册器
func InitMetrics(registerer prometheus.Registerer) {
	if registerer == nil {
		registerer = prometheus.DefaultRegisterer
	}
	registerer.MustRegister(HttpRequestsTotal, HttpRequestDuration)
}
