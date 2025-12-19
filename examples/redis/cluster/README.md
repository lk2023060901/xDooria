# Redis Cluster 集群测试环境

## 架构

```
Redis Cluster (6 节点: 3 主 + 3 从)
├── Master 1 (Node 1) - Port 7001
│   └── Replica 1 (Node 4) - Port 7004
├── Master 2 (Node 2) - Port 7002
│   └── Replica 2 (Node 5) - Port 7005
└── Master 3 (Node 3) - Port 7003
    └── Replica 3 (Node 6) - Port 7006

Hash Slots 分布:
├── Master 1: Slots 0-5460
├── Master 2: Slots 5461-10922
└── Master 3: Slots 10923-16383
```

## 启动

```bash
# 启动 Redis Cluster
docker compose up -d

# 查看日志（集群初始化过程）
docker compose logs redis-cluster-init

# 查看所有节点状态
docker compose ps
```

启动过程：
1. 启动 6 个 Redis 节点（集群模式）
2. 等待所有节点健康检查通过
3. 自动执行集群初始化（3 主 3 从，每个主节点 1 个副本）

## 验证集群状态

### 1. 检查集群信息

```bash
docker exec -it xdooria-redis-cluster-node-1 redis-cli cluster info
```

输出示例：
```
cluster_state:ok
cluster_slots_assigned:16384
cluster_slots_ok:16384
cluster_slots_pfail:0
cluster_slots_fail:0
cluster_known_nodes:6
cluster_size:3
```

### 2. 查看节点信息

```bash
docker exec -it xdooria-redis-cluster-node-1 redis-cli cluster nodes
```

输出示例：
```
a1b2c3d4... redis-node-1:6379@16379 myself,master - 0 1234567890 1 connected 0-5460
e5f6g7h8... redis-node-4:6379@16379 slave a1b2c3d4... 0 1234567890 4 connected
...
```

### 3. 测试数据分片

```bash
# 连接到集群（需要 -c 参数启用集群模式）
docker exec -it xdooria-redis-cluster-node-1 redis-cli -c

# 设置不同的键（会自动路由到不同节点）
127.0.0.1:6379> SET user:1001 "Alice"
-> Redirected to slot [9619] located at redis-node-2:6379
OK

127.0.0.1:6379> SET user:1002 "Bob"
-> Redirected to slot [3096] located at redis-node-1:6379
OK

127.0.0.1:6379> SET user:1003 "Charlie"
-> Redirected to slot [1209] located at redis-node-1:6379
OK

# 查看键所在的 slot
127.0.0.1:6379> CLUSTER KEYSLOT user:1001
(integer) 9619

127.0.0.1:6379> CLUSTER KEYSLOT user:1002
(integer) 3096
```

### 4. 测试 Hash Tags（确保键在同一 slot）

```bash
# 使用 Hash Tags 确保相关键在同一 slot
127.0.0.1:6379> SET player:{p123}:info "player info"
OK

127.0.0.1:6379> SET player:{p123}:bag "bag data"
OK

127.0.0.1:6379> CLUSTER KEYSLOT player:{p123}:info
(integer) 5798

127.0.0.1:6379> CLUSTER KEYSLOT player:{p123}:bag
(integer) 5798

# 两个键在同一 slot，可以在事务中操作
127.0.0.1:6379> MULTI
OK
127.0.0.1:6379> SET player:{p123}:level "10"
QUEUED
127.0.0.1:6379> SET player:{p123}:exp "5000"
QUEUED
127.0.0.1:6379> EXEC
1) OK
2) OK
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
        Cluster: &redis.ClusterConfig{
            Addrs: []string{
                "localhost:7001",
                "localhost:7002",
                "localhost:7003",
                "localhost:7004",
                "localhost:7005",
                "localhost:7006",
            },
            Password: "",
        },
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

    // 测试连接
    if err := client.Ping(ctx); err != nil {
        panic(err)
    }

    // 普通键（会分散到不同 slot）
    if err := client.Set(ctx, "key1", "value1", 0); err != nil {
        panic(err)
    }

    // 使用 Hash Tags（确保同一玩家数据在同一 slot）
    playerID := "p12345"
    keyPrefix := fmt.Sprintf("player:{%s}", playerID)

    // 这些操作可以在同一事务中执行
    pipe := client.Pipeline()
    pipe.Set(keyPrefix+":info", "player info", 0)
    pipe.Set(keyPrefix+":level", "10", 0)
    pipe.Set(keyPrefix+":exp", "5000", 0)
    results, err := pipe.Exec(ctx)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Pipeline executed: %d commands\n", len(results))

    // 读取数据
    info, err := client.Get(ctx, keyPrefix+":info")
    if err != nil {
        panic(err)
    }
    fmt.Println("Player info:", info)
}
```

## 故障测试

### 测试主节点故障（自动故障转移）

```bash
# 停止主节点 1
docker compose stop redis-node-1

# 等待几秒，检查集群状态
docker exec -it redis-cluster-node-2 redis-cli cluster nodes

# 会看到 node-4 (原 node-1 的副本) 被提升为主节点
# 集群仍可正常工作

# 恢复 node-1（会自动成为 node-4 的副本）
docker compose start redis-node-1
```

### 测试从节点故障

```bash
# 停止从节点 4
docker compose stop redis-node-4

# 集群仍可正常工作（主节点 1 仍在）
# 只是失去了一个副本

# 恢复从节点 4
docker compose start redis-node-4
```

### 测试脑裂（停止多个主节点）

```bash
# 停止 2 个主节点（超过半数）
docker compose stop redis-node-1 redis-node-2

# 检查集群状态（会显示 cluster_state:fail）
docker exec -it redis-cluster-node-3 redis-cli cluster info

# 集群进入失败状态（无法提供服务）

# 恢复节点
docker compose start redis-node-1 redis-node-2
```

## 扩容测试

### 添加新节点

```bash
# 启动新节点（手动运行容器）
docker run -d \
  --name redis-cluster-node-7 \
  --network redis_redis-cluster-net \
  -p 7007:6379 \
  -p 17007:16379 \
  redis:7.4-alpine \
  redis-server \
  --port 6379 \
  --cluster-enabled yes \
  --cluster-config-file nodes.conf \
  --cluster-node-timeout 5000 \
  --appendonly no \
  --save ""

# 添加到集群（作为主节点）
docker exec -it redis-cluster-node-1 redis-cli \
  --cluster add-node redis-cluster-node-7:6379 redis-cluster-node-1:6379

# 重新分配 slots
docker exec -it redis-cluster-node-1 redis-cli \
  --cluster reshard redis-cluster-node-1:6379
```

## 性能测试

```bash
# 使用 redis-benchmark 测试集群性能
docker exec -it xdooria-redis-cluster-node-1 redis-benchmark \
  -c 50 \
  -n 10000 \
  -t set,get \
  -q

# 输出示例:
# SET: 45000.00 requests per second
# GET: 50000.00 requests per second
```

## 监控

### 实时监控所有节点

```bash
# 终端 1: 监控 node-1
docker exec -it xdooria-redis-cluster-node-1 redis-cli MONITOR

# 终端 2: 监控 node-2
docker exec -it xdooria-redis-cluster-node-2 redis-cli MONITOR

# 终端 3: 监控 node-3
docker exec -it xdooria-redis-cluster-node-3 redis-cli MONITOR
```

### 查看统计信息

```bash
# 查看节点信息
docker exec -it xdooria-redis-cluster-node-1 redis-cli INFO stats

# 查看慢日志
docker exec -it xdooria-redis-cluster-node-1 redis-cli SLOWLOG GET 10

# 查看客户端连接
docker exec -it xdooria-redis-cluster-node-1 redis-cli CLIENT LIST
```

## 停止

```bash
# 停止并删除容器
docker compose down

# 停止但保留容器
docker compose stop
```

## 适用场景

- ✅ 大数据量（数据分片）
- ✅ 高并发读写
- ✅ 高可用（自动故障转移）
- ✅ 水平扩展（在线扩容）
- ⚠️ 跨 slot 操作需使用 Hash Tags
- ❌ 不支持多数据库（只能使用 db0）

## Hash Tags 最佳实践

### 游戏场景示例

```go
// 玩家维度分片
playerID := "p12345"
keyPrefix := fmt.Sprintf("player:{%s}", playerID)

client.Set(ctx, keyPrefix+":info", playerData, 0)
client.Set(ctx, keyPrefix+":bag", bagData, 0)
client.HSet(ctx, keyPrefix+":stats", "level", 10)

// 公会维度分片
guildID := "g6789"
guildPrefix := fmt.Sprintf("guild:{%s}", guildID)

client.Set(ctx, guildPrefix+":info", guildInfo, 0)
client.SAdd(ctx, guildPrefix+":members", member1, member2)

// 全局数据（不使用 Hash Tags，允许分散）
client.ZAdd(ctx, "global:leaderboard:level",
    redis.ZItem{Member: "player1", Score: 100},
    redis.ZItem{Member: "player2", Score: 95},
)
```

## 常见问题

### Q: 为什么需要 16379 端口？
A: 16379 是集群总线端口（cluster bus port），用于节点间通信、故障检测和配置传播。

### Q: 集群最少需要几个节点？
A: 最少 3 个主节点（官方推荐 3 主 3 从，共 6 节点）。

### Q: 如何查看某个键在哪个节点？
```bash
docker exec -it xdooria-redis-cluster-node-1 redis-cli -c
> CLUSTER KEYSLOT mykey
(integer) 14687
> CLUSTER NODES | grep 14687
```

### Q: 跨 slot 操作报错怎么办？
A: 使用 Hash Tags 确保相关键在同一 slot，例如 `player:{p123}:info` 和 `player:{p123}:bag`。
