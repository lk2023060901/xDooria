package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
)

// NewCounter 创建并注册 Counter
func (c *Client) NewCounter(name, help string, labels []string) (*CounterVec, error) {
	if c.IsClosed() {
		return nil, ErrClientClosed
	}

	// 检查是否已存在
	if _, loaded := c.counters.LoadOrStore(name, nil); loaded {
		return nil, ErrMetricExists
	}

	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: c.config.Namespace,
			Subsystem: c.config.Subsystem,
			Name:      name,
			Help:      help,
		},
		labels,
	)

	if err := c.registry.Register(counter); err != nil {
		c.counters.Delete(name)
		return nil, err
	}

	c.counters.Store(name, counter)
	return counter, nil
}

// MustNewCounter 创建 Counter，失败则 panic
func (c *Client) MustNewCounter(name, help string, labels []string) *CounterVec {
	counter, err := c.NewCounter(name, help, labels)
	if err != nil {
		panic(err)
	}
	return counter
}

// GetCounter 获取已注册的 Counter
func (c *Client) GetCounter(name string) (*CounterVec, bool) {
	v, ok := c.counters.Load(name)
	if !ok {
		return nil, false
	}
	return v.(*CounterVec), true
}

// NewGauge 创建并注册 Gauge
func (c *Client) NewGauge(name, help string, labels []string) (*GaugeVec, error) {
	if c.IsClosed() {
		return nil, ErrClientClosed
	}

	if _, loaded := c.gauges.LoadOrStore(name, nil); loaded {
		return nil, ErrMetricExists
	}

	gauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: c.config.Namespace,
			Subsystem: c.config.Subsystem,
			Name:      name,
			Help:      help,
		},
		labels,
	)

	if err := c.registry.Register(gauge); err != nil {
		c.gauges.Delete(name)
		return nil, err
	}

	c.gauges.Store(name, gauge)
	return gauge, nil
}

// MustNewGauge 创建 Gauge，失败则 panic
func (c *Client) MustNewGauge(name, help string, labels []string) *GaugeVec {
	gauge, err := c.NewGauge(name, help, labels)
	if err != nil {
		panic(err)
	}
	return gauge
}

// GetGauge 获取已注册的 Gauge
func (c *Client) GetGauge(name string) (*GaugeVec, bool) {
	v, ok := c.gauges.Load(name)
	if !ok {
		return nil, false
	}
	return v.(*GaugeVec), true
}

// NewHistogram 创建并注册 Histogram
func (c *Client) NewHistogram(name, help string, labels []string, buckets []float64) (*HistogramVec, error) {
	if c.IsClosed() {
		return nil, ErrClientClosed
	}

	if _, loaded := c.histograms.LoadOrStore(name, nil); loaded {
		return nil, ErrMetricExists
	}

	if buckets == nil {
		buckets = prometheus.DefBuckets
	}

	histogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: c.config.Namespace,
			Subsystem: c.config.Subsystem,
			Name:      name,
			Help:      help,
			Buckets:   buckets,
		},
		labels,
	)

	if err := c.registry.Register(histogram); err != nil {
		c.histograms.Delete(name)
		return nil, err
	}

	c.histograms.Store(name, histogram)
	return histogram, nil
}

// MustNewHistogram 创建 Histogram，失败则 panic
func (c *Client) MustNewHistogram(name, help string, labels []string, buckets []float64) *HistogramVec {
	histogram, err := c.NewHistogram(name, help, labels, buckets)
	if err != nil {
		panic(err)
	}
	return histogram
}

// GetHistogram 获取已注册的 Histogram
func (c *Client) GetHistogram(name string) (*HistogramVec, bool) {
	v, ok := c.histograms.Load(name)
	if !ok {
		return nil, false
	}
	return v.(*HistogramVec), true
}

// NewSummary 创建并注册 Summary
func (c *Client) NewSummary(name, help string, labels []string, objectives map[float64]float64) (*SummaryVec, error) {
	if c.IsClosed() {
		return nil, ErrClientClosed
	}

	if _, loaded := c.summaries.LoadOrStore(name, nil); loaded {
		return nil, ErrMetricExists
	}

	if objectives == nil {
		objectives = map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001}
	}

	summary := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  c.config.Namespace,
			Subsystem:  c.config.Subsystem,
			Name:       name,
			Help:       help,
			Objectives: objectives,
		},
		labels,
	)

	if err := c.registry.Register(summary); err != nil {
		c.summaries.Delete(name)
		return nil, err
	}

	c.summaries.Store(name, summary)
	return summary, nil
}

// MustNewSummary 创建 Summary，失败则 panic
func (c *Client) MustNewSummary(name, help string, labels []string, objectives map[float64]float64) *SummaryVec {
	summary, err := c.NewSummary(name, help, labels, objectives)
	if err != nil {
		panic(err)
	}
	return summary
}

// GetSummary 获取已注册的 Summary
func (c *Client) GetSummary(name string) (*SummaryVec, bool) {
	v, ok := c.summaries.Load(name)
	if !ok {
		return nil, false
	}
	return v.(*SummaryVec), true
}

// RegisterCollector 注册自定义采集器
func (c *Client) RegisterCollector(collector Collector) error {
	if c.IsClosed() {
		return ErrClientClosed
	}
	return c.registry.Register(collector)
}

// MustRegisterCollector 注册自定义采集器，失败则 panic
func (c *Client) MustRegisterCollector(collector Collector) {
	if err := c.RegisterCollector(collector); err != nil {
		panic(err)
	}
}
