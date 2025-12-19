package postgres

import (
	"context"
	"testing"
	"time"
)

// 测试配置
var (
	standaloneConfig = &Config{
		Standalone: &DBConfig{
			Host:     "localhost",
			Port:     25432,
			User:     "xdooria",
			Password: "xdooria_pass",
			DBName:   "xdooria_test",
			SSLMode:  "disable",
		},
		Pool:           getTestPoolConfig(),
		ConnectTimeout: 5 * time.Second,
		QueryTimeout:   30 * time.Second,
	}

	masterSlaveConfig = &Config{
		Master: &DBConfig{
			Host:     "localhost",
			Port:     25432,
			User:     "xdooria",
			Password: "xdooria_pass",
			DBName:   "xdooria_test",
			SSLMode:  "disable",
		},
		Slaves: []DBConfig{
			{
				Host:     "localhost",
				Port:     25433,
				User:     "xdooria",
				Password: "xdooria_pass",
				DBName:   "xdooria_test",
				SSLMode:  "disable",
			},
			{
				Host:     "localhost",
				Port:     25434,
				User:     "xdooria",
				Password: "xdooria_pass",
				DBName:   "xdooria_test",
				SSLMode:  "disable",
			},
		},
		SlaveLoadBalance: "round_robin",
		Pool:             getTestPoolConfig(),
		ConnectTimeout:   5 * time.Second,
		QueryTimeout:     30 * time.Second,
	}
)

func getTestPoolConfig() PoolConfig {
	return PoolConfig{
		MaxConns:          50,
		MinConns:          5,
		MaxConnLifetime:   1 * time.Hour,
		MaxConnIdleTime:   10 * time.Minute,
		HealthCheckPeriod: 30 * time.Second,
	}
}

// TestConfigValidation 测试配置验证
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "valid standalone",
			config: &Config{
				Standalone: &DBConfig{
					Host:     "localhost",
					Port:     25432,
					User:     "xdooria",
					Password: "xdooria_pass",
					DBName:   "xdooria_test",
				},
				Pool: getTestPoolConfig(),
			},
			wantErr: false,
		},
		{
			name: "valid master-slave",
			config: &Config{
				Master: &DBConfig{
					Host:     "localhost",
					Port:     25432,
					User:     "xdooria",
					Password: "xdooria_pass",
					DBName:   "xdooria_test",
				},
				Slaves: []DBConfig{
					{
						Host:     "localhost",
						Port:     25433,
						User:     "xdooria",
						Password: "xdooria_pass",
						DBName:   "xdooria_test",
					},
				},
				Pool: getTestPoolConfig(),
			},
			wantErr: false,
		},
		{
			name: "multiple modes configured",
			config: &Config{
				Standalone: &DBConfig{Host: "localhost", Port: 25432, User: "test", Password: "test", DBName:   "test"},
				Master:     &DBConfig{Host: "localhost", Port: 25432, User: "test", Password: "test", DBName:   "test"},
				Pool:       getTestPoolConfig(),
			},
			wantErr: true,
		},
		{
			name:    "no mode configured",
			config:  &Config{Pool: getTestPoolConfig()},
			wantErr: true,
		},
		{
			name: "invalid slave load balance",
			config: &Config{
				Master: &DBConfig{
					Host:     "localhost",
					Port:     25432,
					User:     "xdooria",
					Password: "xdooria_pass",
					DBName:   "xdooria_test",
				},
				Slaves: []DBConfig{
					{Host: "localhost", Port: 25433, User: "test", Password: "test", DBName:   "test"},
				},
				SlaveLoadBalance: "invalid",
				Pool:             getTestPoolConfig(),
			},
			wantErr: true,
		},
		{
			name: "missing required fields",
			config: &Config{
				Standalone: &DBConfig{
					Host: "localhost",
					Port: 25432,
					// Missing User, Password, Database
				},
				Pool: getTestPoolConfig(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestNewClientStandalone 测试创建 Standalone 客户端
func TestNewClientStandalone(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := New(standaloneConfig)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 测试连接
	if err := client.Ping(ctx); err != nil {
		t.Errorf("Ping() error = %v", err)
	}

	// 测试连接池统计
	stats := client.Stats()
	if stats == nil {
		t.Error("Stats() should not return nil")
	}
	if stats.TotalConns == 0 {
		t.Error("Stats() TotalConns should not be 0 after Ping()")
	}
}

// TestNewClientMasterSlave 测试创建 Master-Slave 客户端
func TestNewClientMasterSlave(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Note: This test requires the master-slave environment to be running
	// docker compose up -d in examples/postgres/master-slave/
	t.Skip("skipping master-slave test - requires manual setup")

	client, err := New(masterSlaveConfig)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 测试连接（会测试主库和所有从库）
	if err := client.Ping(ctx); err != nil {
		t.Errorf("Ping() error = %v", err)
	}
}

// TestClientClose 测试客户端关闭
func TestClientClose(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := New(standaloneConfig)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx := context.Background()

	// 测试连接
	if err := client.Ping(ctx); err != nil {
		t.Errorf("Ping() error = %v", err)
	}

	// 关闭客户端
	client.Close()

	// 关闭后操作应该失败
	if err := client.Ping(ctx); err == nil {
		t.Error("Ping() after Close() should return error")
	}
}

// TestGetSlaveLoadBalance 测试从库负载均衡策略
func TestGetSlaveLoadBalance(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   string
	}{
		{
			name: "default to random",
			config: &Config{
				SlaveLoadBalance: "",
			},
			want: "random",
		},
		{
			name: "explicit round_robin",
			config: &Config{
				SlaveLoadBalance: "round_robin",
			},
			want: "round_robin",
		},
		{
			name: "explicit random",
			config: &Config{
				SlaveLoadBalance: "random",
			},
			want: "random",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetSlaveLoadBalance()
			if got != tt.want {
				t.Errorf("GetSlaveLoadBalance() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestConfigBuildDSN 测试 DSN 构建
func TestConfigBuildDSN(t *testing.T) {
	tests := []struct {
		name     string
		dbConfig *DBConfig
		want     string
	}{
		{
			name: "basic config",
			dbConfig: &DBConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "testuser",
				Password: "testpass",
				DBName:   "testdb",
				SSLMode:  "disable",
			},
			want: "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable",
		},
		{
			name: "with ssl mode",
			dbConfig: &DBConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "testuser",
				Password: "testpass",
				DBName:   "testdb",
				SSLMode:  "require",
			},
			want: "postgres://testuser:testpass@localhost:5432/testdb?sslmode=require",
		},
		{
			name: "default ssl mode",
			dbConfig: &DBConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "testuser",
				Password: "testpass",
				DBName:   "testdb",
			},
			want: "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.dbConfig.BuildDSN()
			if got != tt.want {
				t.Errorf("BuildDSN() = %v, want %v", got, tt.want)
			}
		})
	}
}

// BenchmarkPing Benchmark Ping 操作
func BenchmarkPing(b *testing.B) {
	client, err := New(standaloneConfig)
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := client.Ping(ctx); err != nil {
			b.Errorf("Ping() error = %v", err)
		}
	}
}
