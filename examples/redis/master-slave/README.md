# Redis Master-Slave 主从测试环境

## 架构

```
Redis Master (主节点)
  ├── Port: 6379
  ├── Role: master
  └── Persistence: 关闭
      ↓ (复制)
      ├── Redis Slave 1 (从节点 1)
      │     ├── Port: 6380
      │     ├── Role: slave
      │     └── Read-only: yes
      │
      └── Redis Slave 2 (从节点 2)
            ├── Port: 6381
            ├── Role: slave
            └── Read-only: yes
```

## 启动

```bash
# 启动 Redis Master-Slave 集群
docker compose up -d

# 查看日志
docker compose logs -f

# 查看状态
docker compose ps
```

## 验证主从复制

### 1. 检查主节点

```bash
docker exec -it xdooria-redis-master redis-cli INFO replication
```

输出示例：
```
# Replication
role:master
connected_slaves:2
slave0:ip=172.18.0.3,port=6379,state=online,offset=...
slave1:ip=172.18.0.4,port=6379,state=online,offset=...
```

### 2. 检查从节点

```bash
docker exec -it xdooria-redis-slave-1 redis-cli INFO replication
```

输出示例：
```
# Replication
role:slave
master_host:redis-master
master_port:6379
master_link_status:up
```

### 3. 测试数据复制

```bash
# 在主节点写入数据
docker exec -it xdooria-redis-master redis-cli SET test:replication "hello from master"

# 在从节点读取数据
docker exec -it xdooria-redis-slave-1 redis-cli GET test:replication
# 输出: "hello from master"

docker exec -it xdooria-redis-slave-2 redis-cli GET test:replication
# 输出: "hello from master"

# 尝试在从节点写入（会失败）
docker exec -it xdooria-redis-slave-1 redis-cli SET test:write "fail"
# 输出: (error) READONLY You can't write against a read only replica.
```

## Go 代码配置

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/lk2023060901/xdooria/pkg/database/redis"
)

func main() {
    cfg := &redis.Config{
        Master: &redis.NodeConfig{
            Host:     "localhost",
            Port:     6379,
            Password: "",
            DB:       0,
        },
        Slaves: []redis.NodeConfig{
            {
                Host:     "localhost",
                Port:     6380,
                Password: "",
                DB:       0,
            },
            {
                Host:     "localhost",
                Port:     6381,
                Password: "",
                DB:       0,
            },
        },
        SlaveLoadBalance: "round_robin", // 或 "random"
        Pool: redis.PoolConfig{
            MaxIdleConns:    10,
            MaxOpenConns:    100,
            ConnMaxLifetime: 1 * time.Hour,
            ConnMaxIdleTime: 10 * time.Minute,
            DialTimeout:     5 * time.Second,
            ReadTimeout:     3 * time.Second,
            WriteTimeout:    3 * time.Second,
            PoolTimeout:     5 * time.Second,
        },
    }

    client, err := redis.NewClient(cfg)
    if err != nil {
        panic(err)
    }
    defer client.Close()

    ctx := context.Background()

    // 测试连接（会测试主库和所有从库）
    if err := client.Ping(ctx); err != nil {
        panic(err)
    }

    // 写操作 -> 主库
    if err := client.Set(ctx, "test:key", "hello", 0); err != nil {
        panic(err)
    }

    // 读操作 -> 从库（负载均衡）
    val, err := client.Get(ctx, "test:key")
    if err != nil {
        panic(err)
    }

    fmt.Println("Value:", val)

    // 查看连接池统计
    stats := client.PoolStats()
    fmt.Printf("Pool Stats: Hits=%d, Misses=%d, TotalConns=%d\n",
        stats.Hits, stats.Misses, stats.TotalConns)
}
```

## 测试负载均衡

```bash
# 监控从节点 1 的命令
docker exec -it xdooria-redis-slave-1 redis-cli MONITOR &

# 监控从节点 2 的命令
docker exec -it xdooria-redis-slave-2 redis-cli MONITOR &

# 在应用中执行多次读操作，观察请求分布
```

## 故障测试

### 测试从节点故障

```bash
# 停止从节点 1
docker compose stop redis-slave-1

# 应用仍可正常工作（读请求会路由到从节点 2）
# 写请求不受影响（仍写入主节点）

# 恢复从节点 1
docker compose start redis-slave-1
```

### 测试主节点故障

```bash
# 停止主节点
docker compose stop redis-master

# 此时：
# - 写操作会失败（无主节点）
# - 读操作仍可用（从节点可读）

# 恢复主节点
docker compose start redis-master
```

## 停止

```bash
# 停止并删除容器
docker compose down

# 停止但保留容器
docker compose stop
```

## 适用场景

- ✅ 读多写少的应用
- ✅ 需要读写分离
- ✅ 提高读性能（从库负载均衡）
- ✅ 数据冗余备份
- ⚠️ 主节点故障需手动切换
- ❌ 不支持自动故障转移（需 Redis Sentinel）

## 负载均衡策略

### Random（随机）
```go
SlaveLoadBalance: "random"
```
- 随机选择从节点
- 分布相对均匀
- 适合大多数场景

### Round Robin（轮询）
```go
SlaveLoadBalance: "round_robin"
```
- 按顺序轮询从节点
- 分布绝对均匀
- 适合从节点性能一致的场景
