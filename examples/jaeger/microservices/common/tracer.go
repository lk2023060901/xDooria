package common

import (
	"fmt"
	"os"
	"time"

	"github.com/lk2023060901/xdooria/pkg/jaeger"
)

// InitTracer 初始化 Jaeger Tracer
func InitTracer(serviceName string) (*jaeger.Tracer, error) {
	endpoint := os.Getenv("JAEGER_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:4318"
	}

	cfg := &jaeger.Config{
		Enabled:     true,
		ServiceName: serviceName,
		Endpoint:    endpoint,
		Sampler: jaeger.SamplerConfig{
			Type:  jaeger.SamplerTypeAlways,
			Ratio: 1.0,
		},
		Attributes: map[string]string{
			"environment": "dev",
			"version":     "1.0.0",
		},
		BatchExport: jaeger.BatchExportConfig{
			BatchSize:     512,
			ExportTimeout: 30 * time.Second,
			MaxQueueSize:  2048,
			BatchTimeout:  5 * time.Second,
		},
		ShutdownTimeout: 5 * time.Second,
	}

	tracer, err := jaeger.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create tracer: %w", err)
	}

	return tracer, nil
}
