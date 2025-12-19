# Redis 测试环境

本目录提供三种 Redis 部署模式的 Docker Compose 测试环境，用于本地开发和测试 xDooria Redis 封装。

## 目录结构

```
examples/redis/
├── README.md                    # 本文件
├── standalone/                  # 单机模式
│   ├── docker-compose.yaml
│   └── README.md
├── master-slave/                # 主从模式
│   ├── docker-compose.yaml
│   └── README.md
└── cluster/                     # 集群模式
    ├── docker-compose.yaml
    └── README.md
```

## 三种模式对比

| 特性 | Standalone | Master-Slave | Cluster |
|------|-----------|--------------|---------|
| **节点数** | 1 | 1 主 + 2 从 | 3 主 + 3 从 |
| **端口** | 6379 | 6379, 6380, 6381 | 7001-7006 |
| **高可用** | ❌ | ⚠️ 手动切换 | ✅ 自动故障转移 |
| **读写分离** | ❌ | ✅ | ✅ |
| **数据分片** | ❌ | ❌ | ✅ (16384 slots) |
| **水平扩展** | ❌ | ❌ | ✅ |
| **负载均衡** | ❌ | ✅ (random/round_robin) | ✅ (自动) |
| **适用场景** | 开发测试 | 读多写少 | 生产环境 |
| **复杂度** | ⭐ | ⭐⭐ | ⭐⭐⭐ |

## 快速开始

### 1. Standalone（单机模式）

**适用场景**: 本地开发、简单测试

```bash
cd standalone
docker compose up -d
```

**配置示例**:
```go
cfg := &redis.Config{
    Standalone: &redis.NodeConfig{
        Host: "localhost",
        Port: 6379,
    },
    Pool: redis.PoolConfig{ /* ... */ },
}
```

### 2. Master-Slave（主从模式）

**适用场景**: 读多写少、需要读写分离

```bash
cd master-slave
docker compose up -d
```

**配置示例**:
```go
cfg := &redis.Config{
    Master: &redis.NodeConfig{
        Host: "localhost",
        Port: 6379,
    },
    Slaves: []redis.NodeConfig{
        {Host: "localhost", Port: 6380},
        {Host: "localhost", Port: 6381},
    },
    SlaveLoadBalance: "round_robin",
    Pool: redis.PoolConfig{ /* ... */ },
}
```

### 3. Cluster（集群模式）

**适用场景**: 大数据量、高并发、生产环境

```bash
cd cluster
docker compose up -d

# 等待集群初始化完成（约 10-15 秒）
docker compose logs redis-cluster-init
```

**配置示例**:
```go
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
    },
    Pool: redis.PoolConfig{ /* ... */ },
}
```

## 通用操作

### 查看运行状态

```bash
# 查看容器状态
docker compose ps

# 查看日志
docker compose logs -f

# 查看特定容器日志
docker compose logs -f <service-name>
```

### 连接 Redis

```bash
# Standalone
docker exec -it xdooria-redis-standalone redis-cli

# Master-Slave
docker exec -it xdooria-redis-master redis-cli         # 主节点
docker exec -it xdooria-redis-slave-1 redis-cli        # 从节点 1

# Cluster (需要 -c 参数启用集群模式)
docker exec -it xdooria-redis-cluster-node-1 redis-cli -c
```

### 停止环境

```bash
# 停止并删除容器（推荐）
docker compose down

# 仅停止容器（保留配置）
docker compose stop

# 重启
docker compose restart
```

### 清理资源

```bash
# 删除所有 Redis 容器和网络
docker compose down

# 删除未使用的网络
docker network prune -f
```

## 测试示例

### 完整测试代码

创建 `test.go`:

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/lk2023060901/xdooria/pkg/database/redis"
)

func main() {
    // 根据需要选择配置
    cfg := getStandaloneConfig()  // 或 getMasterSlaveConfig() 或 getClusterConfig()

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
    fmt.Println("✅ 连接成功")

    // 测试基本操作
    testBasicOperations(client, ctx)

    // 测试对象序列化
    testObjectSerialization(client, ctx)

    // 测试 Pipeline
    testPipeline(client, ctx)

    // 测试分布式锁
    testLock(client, ctx)

    fmt.Println("✅ 所有测试通过")
}

func getStandaloneConfig() *redis.Config {
    return &redis.Config{
        Standalone: &redis.NodeConfig{
            Host: "localhost",
            Port: 6379,
        },
        Pool: getPoolConfig(),
    }
}

func getMasterSlaveConfig() *redis.Config {
    return &redis.Config{
        Master: &redis.NodeConfig{
            Host: "localhost",
            Port: 6379,
        },
        Slaves: []redis.NodeConfig{
            {Host: "localhost", Port: 6380},
            {Host: "localhost", Port: 6381},
        },
        SlaveLoadBalance: "round_robin",
        Pool: getPoolConfig(),
    }
}

func getClusterConfig() *redis.Config {
    return &redis.Config{
        Cluster: &redis.ClusterConfig{
            Addrs: []string{
                "localhost:7001",
                "localhost:7002",
                "localhost:7003",
            },
        },
        Pool: getPoolConfig(),
    }
}

func getPoolConfig() redis.PoolConfig {
    return redis.PoolConfig{
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

func testBasicOperations(client *redis.Client, ctx context.Context) {
    fmt.Println("\n📝 测试基本操作...")

    // String 操作
    client.Set(ctx, "test:string", "hello", 10*time.Second)
    val, _ := client.Get(ctx, "test:string")
    fmt.Printf("  String: %s\n", val)

    // Hash 操作
    client.HSet(ctx, "test:hash", "field1", "value1", "field2", "value2")
    hashVal, _ := client.HGetAll(ctx, "test:hash")
    fmt.Printf("  Hash: %v\n", hashVal)

    // List 操作
    client.RPush(ctx, "test:list", "item1", "item2", "item3")
    listVal, _ := client.LRange(ctx, "test:list", 0, -1)
    fmt.Printf("  List: %v\n", listVal)

    // Set 操作
    client.SAdd(ctx, "test:set", "member1", "member2", "member3")
    setVal, _ := client.SMembers(ctx, "test:set")
    fmt.Printf("  Set: %v\n", setVal)

    // Sorted Set 操作
    client.ZAdd(ctx, "test:zset",
        redis.ZItem{Member: "player1", Score: 100},
        redis.ZItem{Member: "player2", Score: 95},
    )
    zsetVal, _ := client.ZRevRangeWithScores(ctx, "test:zset", 0, -1)
    fmt.Printf("  ZSet: %v\n", zsetVal)
}

type Player struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Level int    `json:"level"`
}

func testObjectSerialization(client *redis.Client, ctx context.Context) {
    fmt.Println("\n🔄 测试对象序列化...")

    player := &Player{
        ID:    "p12345",
        Name:  "Alice",
        Level: 10,
    }

    // 保存对象
    redis.SetObject(client, ctx, "test:player", player, 10*time.Second)

    // 读取对象
    retrieved, err := redis.GetObject[Player](client, ctx, "test:player")
    if err != nil {
        panic(err)
    }

    fmt.Printf("  Player: %+v\n", retrieved)
}

func testPipeline(client *redis.Client, ctx context.Context) {
    fmt.Println("\n⚡ 测试 Pipeline...")

    results, err := client.Pipelined(ctx, func(p *redis.Pipeline) error {
        p.Set("test:pipe:1", "value1", 0)
        p.Set("test:pipe:2", "value2", 0)
        p.Set("test:pipe:3", "value3", 0)
        p.Incr("test:pipe:counter")
        p.Incr("test:pipe:counter")
        return nil
    })

    if err != nil {
        panic(err)
    }

    fmt.Printf("  Pipeline 执行: %d 个命令\n", len(results))
}

func testLock(client *redis.Client, ctx context.Context) {
    fmt.Println("\n🔒 测试分布式锁...")

    err := client.WithLock(ctx, "test:lock:resource", 5*time.Second, func() error {
        fmt.Println("  获取锁成功，执行业务逻辑...")
        time.Sleep(1 * time.Second)
        return nil
    })

    if err != nil {
        panic(err)
    }

    fmt.Println("  锁已释放")
}
```

### 运行测试

```bash
# 启动对应的 Redis 环境
cd standalone  # 或 master-slave 或 cluster
docker compose up -d

# 运行测试
cd ../..
go run examples/redis/test.go
```

## 常见问题

### Q: 端口冲突怎么办？

如果本地已有 Redis 服务占用 6379 端口：

```yaml
# 修改 docker-compose.yaml 中的端口映射
ports:
  - "6380:6379"  # 改为其他端口
```

### Q: 如何查看 Redis 内存使用？

```bash
docker exec -it <container-name> redis-cli INFO memory
```

### Q: 如何清空所有数据？

```bash
docker exec -it <container-name> redis-cli FLUSHALL
```

### Q: Cluster 模式启动失败？

检查集群初始化日志：
```bash
docker compose logs redis-cluster-init
```

常见原因：
- 节点未完全启动（等待 10-15 秒）
- 端口被占用
- 内存不足

### Q: 如何监控 Redis 性能？

```bash
# 实时监控命令
docker exec -it <container-name> redis-cli MONITOR

# 查看慢日志
docker exec -it <container-name> redis-cli SLOWLOG GET 10

# 统计信息
docker exec -it <container-name> redis-cli INFO stats
```

## 性能基准测试

```bash
# Standalone
docker exec -it redis-standalone redis-benchmark -q -t set,get -n 100000

# Cluster
docker exec -it redis-cluster-node-1 redis-benchmark -c 50 -n 100000 -t set,get -q
```

## 下一步

1. 阅读各模式的详细文档：
   - [Standalone 详细文档](standalone/README.md)
   - [Master-Slave 详细文档](master-slave/README.md)
   - [Cluster 详细文档](cluster/README.md)

2. 查看 Redis 封装 API 文档：
   - [pkg/database/redis](../../pkg/database/redis/)

3. 参考技术栈文档：
   - [docs/tech-stack.md](../../docs/tech-stack.md)
   - [docs/infrastructure.md](../../docs/infrastructure.md)

## 参考资源

- [Redis 官方文档](https://redis.io/documentation)
- [Redis Cluster 教程](https://redis.io/docs/manual/scaling/)
- [go-redis 文档](https://redis.uptrace.dev/)
