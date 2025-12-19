# Redis Standalone 测试环境

## 架构

```
Redis Standalone (单节点)
  ├── Port: 6379
  ├── MaxMemory: 256MB
  └── Persistence: 关闭
```

## 启动

```bash
# 启动 Redis Standalone
docker compose up -d

# 查看日志
docker compose logs -f

# 查看状态
docker compose ps
```

## 连接测试

```bash
# 使用 redis-cli 连接
docker exec -it xdooria-redis-standalone redis-cli

# 测试命令
> PING
PONG

> SET test:key "hello"
OK

> GET test:key
"hello"

> INFO replication
# Replication
role:master
connected_slaves:0
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
        Standalone: &redis.NodeConfig{
            Host:     "localhost",
            Port:     6379,
            Password: "",
            DB:       0,
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

    // 设置值
    if err := client.Set(ctx, "test:key", "hello", 0); err != nil {
        panic(err)
    }

    // 获取值
    val, err := client.Get(ctx, "test:key")
    if err != nil {
        panic(err)
    }

    fmt.Println("Value:", val)
}
```

## 停止

```bash
# 停止并删除容器
docker compose down

# 停止但保留容器
docker compose stop
```

## 适用场景

- ✅ 开发环境测试
- ✅ 单机应用
- ✅ 低并发场景
- ❌ 不适合生产环境高可用需求
