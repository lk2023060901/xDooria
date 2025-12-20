package postgres

import (
	"context"
	"testing"
	"time"
)

// TestApplyQueryTimeout 测试查询超时应用
func TestApplyQueryTimeout(t *testing.T) {
	tests := []struct {
		name           string
		queryTimeout   time.Duration
		wantTimeout    bool
		expectedMinDur time.Duration
	}{
		{
			name:         "with query timeout",
			queryTimeout: 5 * time.Second,
			wantTimeout:  true,
		},
		{
			name:         "without query timeout",
			queryTimeout: 0,
			wantTimeout:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建一个模拟的客户端配置
			cfg := &Config{
				QueryTimeout: tt.queryTimeout,
			}

			client := &Client{
				cfg: cfg,
			}

			// 创建一个基础 context
			baseCtx := context.Background()

			// 应用超时
			newCtx, cancel := client.applyQueryTimeout(baseCtx)
			defer cancel()

			// 检查是否有超时
			deadline, hasDeadline := newCtx.Deadline()

			if tt.wantTimeout {
				if !hasDeadline {
					t.Error("Expected context to have deadline, but it doesn't")
					return
				}

				// 验证超时时间大致正确（允许一些误差）
				expectedDeadline := time.Now().Add(tt.queryTimeout)
				timeDiff := expectedDeadline.Sub(deadline)
				if timeDiff < -100*time.Millisecond || timeDiff > 100*time.Millisecond {
					t.Errorf("Deadline difference too large: %v", timeDiff)
				}
			} else {
				if hasDeadline {
					t.Error("Expected context to not have deadline, but it does")
				}
			}
		})
	}
}

// TestTxWrapperApplyQueryTimeout 测试事务超时应用
func TestTxWrapperApplyQueryTimeout(t *testing.T) {
	tests := []struct {
		name        string
		timeout     time.Duration
		wantTimeout bool
	}{
		{
			name:        "with timeout",
			timeout:     3 * time.Second,
			wantTimeout: true,
		},
		{
			name:        "without timeout",
			timeout:     0,
			wantTimeout: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建一个模拟的事务包装器
			tx := &txWrapper{
				timeout: tt.timeout,
			}

			// 创建一个基础 context
			baseCtx := context.Background()

			// 应用超时
			newCtx, cancel := tx.applyQueryTimeout(baseCtx)
			defer cancel()

			// 检查是否有超时
			_, hasDeadline := newCtx.Deadline()

			if tt.wantTimeout && !hasDeadline {
				t.Error("Expected context to have deadline, but it doesn't")
			}

			if !tt.wantTimeout && hasDeadline {
				t.Error("Expected context to not have deadline, but it does")
			}
		})
	}
}

// TestConfigQueryTimeout 测试配置中的 QueryTimeout 字段
func TestConfigQueryTimeout(t *testing.T) {
	cfg := &Config{
		Standalone: &DBConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "test",
			Password: "test",
			DBName:   "test",
		},
		Pool:         getTestPoolConfig(),
		QueryTimeout: 10 * time.Second,
	}

	if cfg.QueryTimeout != 10*time.Second {
		t.Errorf("Expected QueryTimeout to be 10s, got %v", cfg.QueryTimeout)
	}
}
