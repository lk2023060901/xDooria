package redis

import (
	"context"
	"testing"
	"time"
)

// 测试配置
var (
	standaloneConfig = &Config{
		Standalone: &NodeConfig{
			Host:     "localhost",
			Port:     16379,
			Password: "",
			DB:       0,
		},
		Pool: getTestPoolConfig(),
	}

	masterSlaveConfig = &Config{
		Master: &NodeConfig{
			Host:     "localhost",
			Port:     16379,
			Password: "",
			DB:       0,
		},
		Slaves: []NodeConfig{
			{Host: "localhost", Port: 16380, Password: "", DB: 0},
			{Host: "localhost", Port: 16381, Password: "", DB: 0},
		},
		SlaveLoadBalance: "round_robin",
		Pool:             getTestPoolConfig(),
	}

	clusterConfig = &Config{
		Cluster: &ClusterConfig{
			Addrs: []string{
				"localhost:17001",
				"localhost:17002",
				"localhost:17003",
			},
			Password: "",
		},
		Pool: getTestPoolConfig(),
	}
)

func getTestPoolConfig() PoolConfig {
	return PoolConfig{
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: 1 * time.Hour,
		ConnMaxIdleTime: 10 * time.Minute,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolTimeout:     5 * time.Second,
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
				Standalone: &NodeConfig{Host: "localhost", Port: 16379},
				Pool:       getTestPoolConfig(),
			},
			wantErr: false,
		},
		{
			name: "valid master-slave",
			config: &Config{
				Master: &NodeConfig{Host: "localhost", Port: 16379},
				Slaves: []NodeConfig{
					{Host: "localhost", Port: 16380},
				},
				Pool: getTestPoolConfig(),
			},
			wantErr: false,
		},
		{
			name: "valid cluster",
			config: &Config{
				Cluster: &ClusterConfig{
					Addrs: []string{"localhost:17001"},
				},
				Pool: getTestPoolConfig(),
			},
			wantErr: false,
		},
		{
			name: "multiple modes configured",
			config: &Config{
				Standalone: &NodeConfig{Host: "localhost", Port: 16379},
				Master:     &NodeConfig{Host: "localhost", Port: 16379},
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
				Master: &NodeConfig{Host: "localhost", Port: 16379},
				Slaves: []NodeConfig{
					{Host: "localhost", Port: 16380},
				},
				SlaveLoadBalance: "invalid",
				Pool:             getTestPoolConfig(),
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

	client, err := NewClient(standaloneConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 测试连接
	if err := client.Ping(ctx); err != nil {
		t.Errorf("Ping() error = %v", err)
	}

	// 测试连接池统计
	stats := client.PoolStats()
	if stats.TotalConns == 0 {
		t.Error("PoolStats() TotalConns should not be 0 after Ping()")
	}
}

// TestNewClientMasterSlave 测试创建 Master-Slave 客户端
func TestNewClientMasterSlave(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient(masterSlaveConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 测试连接（会测试主库和所有从库）
	if err := client.Ping(ctx); err != nil {
		t.Errorf("Ping() error = %v", err)
	}
}

// TestNewClientCluster 测试创建 Cluster 客户端
func TestNewClientCluster(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient(clusterConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 测试连接
	if err := client.Ping(ctx); err != nil {
		t.Errorf("Ping() error = %v", err)
	}
}

// TestClientClose 测试客户端关闭
func TestClientClose(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient(standaloneConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()

	// 测试连接
	if err := client.Ping(ctx); err != nil {
		t.Errorf("Ping() error = %v", err)
	}

	// 关闭客户端
	if err := client.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

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

// BenchmarkPing Benchmark Ping 操作
func BenchmarkPing(b *testing.B) {
	client, err := NewClient(standaloneConfig)
	if err != nil {
		b.Fatalf("NewClient() error = %v", err)
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
