package jaeger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.True(t, cfg.Enabled)
	assert.Equal(t, "unknown-service", cfg.ServiceName)
	assert.Equal(t, "http://localhost:14268/api/traces", cfg.Endpoint)
	assert.Equal(t, EndpointTypeCollector, cfg.EndpointType)
	assert.Equal(t, SamplerTypeParent, cfg.Sampler.Type)
	assert.Equal(t, 1.0, cfg.Sampler.Ratio)
	assert.Equal(t, 512, cfg.BatchExport.BatchSize)
	assert.Equal(t, 2048, cfg.BatchExport.MaxQueueSize)
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				Enabled:      true,
				ServiceName:  "test-service",
				Endpoint:     "http://localhost:14268/api/traces",
				EndpointType: EndpointTypeCollector,
				Sampler: SamplerConfig{
					Type:  SamplerTypeRatio,
					Ratio: 0.5,
				},
			},
			wantErr: false,
		},
		{
			name: "disabled config - skip validation",
			cfg: &Config{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "empty service name",
			cfg: &Config{
				Enabled:      true,
				ServiceName:  "",
				Endpoint:     "http://localhost:14268/api/traces",
				EndpointType: EndpointTypeCollector,
			},
			wantErr: true,
		},
		{
			name: "empty endpoint",
			cfg: &Config{
				Enabled:      true,
				ServiceName:  "test-service",
				Endpoint:     "",
				EndpointType: EndpointTypeCollector,
			},
			wantErr: true,
		},
		{
			name: "invalid endpoint type",
			cfg: &Config{
				Enabled:      true,
				ServiceName:  "test-service",
				Endpoint:     "http://localhost:14268/api/traces",
				EndpointType: "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid ratio - negative",
			cfg: &Config{
				Enabled:      true,
				ServiceName:  "test-service",
				Endpoint:     "http://localhost:14268/api/traces",
				EndpointType: EndpointTypeCollector,
				Sampler: SamplerConfig{
					Type:  SamplerTypeRatio,
					Ratio: -0.1,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid ratio - greater than 1",
			cfg: &Config{
				Enabled:      true,
				ServiceName:  "test-service",
				Endpoint:     "http://localhost:14268/api/traces",
				EndpointType: EndpointTypeCollector,
				Sampler: SamplerConfig{
					Type:  SamplerTypeRatio,
					Ratio: 1.5,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewTracerDisabled(t *testing.T) {
	// 注意：MergeConfig 不会用零值覆盖，所以需要使用完整配置来禁用
	cfg := DefaultConfig()
	cfg.Enabled = false

	tracer, err := New(cfg)
	require.NoError(t, err)
	require.NotNil(t, tracer)

	assert.False(t, tracer.IsEnabled())
	assert.False(t, tracer.IsClosed())

	// Close should work
	err = tracer.Close()
	assert.NoError(t, err)
	assert.True(t, tracer.IsClosed())

	// Double close should return error
	err = tracer.Close()
	assert.Equal(t, ErrTracerClosed, err)
}

func TestCreateSampler(t *testing.T) {
	tests := []struct {
		name string
		cfg  SamplerConfig
	}{
		{
			name: "always sampler",
			cfg:  SamplerConfig{Type: SamplerTypeAlways},
		},
		{
			name: "never sampler",
			cfg:  SamplerConfig{Type: SamplerTypeNever},
		},
		{
			name: "ratio sampler",
			cfg:  SamplerConfig{Type: SamplerTypeRatio, Ratio: 0.5},
		},
		{
			name: "parent sampler",
			cfg:  SamplerConfig{Type: SamplerTypeParent},
		},
		{
			name: "default sampler",
			cfg:  SamplerConfig{Type: "unknown"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sampler := createSampler(tt.cfg)
			assert.NotNil(t, sampler)
		})
	}
}

func TestContextFunctions(t *testing.T) {
	ctx := context.Background()

	// No span in context
	assert.Empty(t, TraceIDFromContext(ctx))
	assert.Empty(t, SpanIDFromContext(ctx))
	assert.False(t, IsSampled(ctx))
	assert.False(t, IsRecording(ctx))

	spanCtx := SpanContext(ctx)
	assert.False(t, spanCtx.HasTraceID())
	assert.False(t, spanCtx.HasSpanID())
}

func TestSpanFromContext(t *testing.T) {
	ctx := context.Background()

	span := SpanFromContext(ctx)
	assert.NotNil(t, span)

	// The span should be a no-op span
	assert.False(t, span.IsRecording())
}

func TestContextWithSpan(t *testing.T) {
	ctx := context.Background()
	span := SpanFromContext(ctx)

	newCtx := ContextWithSpan(ctx, span)
	assert.NotNil(t, newCtx)

	retrievedSpan := SpanFromContext(newCtx)
	assert.Equal(t, span, retrievedSpan)
}

func TestTracerWithRealSpan(t *testing.T) {
	// 创建禁用的 tracer（不需要真实的 Jaeger 连接）
	cfg := DefaultConfig()
	cfg.Enabled = false

	tracer, err := New(cfg)
	require.NoError(t, err)
	defer tracer.Close()

	// 使用 noop tracer 创建 span
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	assert.NotNil(t, ctx)
	assert.NotNil(t, span)
}

func TestTracerProvider(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = false

	tracer, err := New(cfg)
	require.NoError(t, err)
	defer tracer.Close()

	provider := tracer.Provider()
	assert.NotNil(t, provider)
}

func TestTracerGetTracer(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = false

	tracer, err := New(cfg)
	require.NoError(t, err)
	defer tracer.Close()

	tr := tracer.Tracer("test-tracer")
	assert.NotNil(t, tr)
}

func TestCreateResource(t *testing.T) {
	cfg := &Config{
		ServiceName: "test-service",
		Attributes: map[string]string{
			"env":     "test",
			"version": "1.0.0",
		},
	}

	res, err := createResource(cfg)
	require.NoError(t, err)
	assert.NotNil(t, res)
}

func TestEndpointTypes(t *testing.T) {
	assert.Equal(t, EndpointType("collector"), EndpointTypeCollector)
	assert.Equal(t, EndpointType("agent"), EndpointTypeAgent)
}

func TestSamplerTypes(t *testing.T) {
	assert.Equal(t, SamplerType("always"), SamplerTypeAlways)
	assert.Equal(t, SamplerType("never"), SamplerTypeNever)
	assert.Equal(t, SamplerType("ratio"), SamplerTypeRatio)
	assert.Equal(t, SamplerType("parent"), SamplerTypeParent)
}

func TestSpanAttributes(t *testing.T) {
	// 测试 attribute 构建
	attrs := []attribute.KeyValue{
		attribute.String("key1", "value1"),
		attribute.Int("key2", 42),
		attribute.Bool("key3", true),
	}

	assert.Len(t, attrs, 3)
}

func TestTracerOptionsPassthrough(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = false

	tracer, err := New(cfg)
	require.NoError(t, err)
	defer tracer.Close()

	// 测试带选项的 Tracer 获取
	tr := tracer.Tracer("test-tracer", trace.WithInstrumentationVersion("1.0.0"))
	assert.NotNil(t, tr)
}
