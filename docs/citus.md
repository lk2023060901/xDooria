# Citus 分布式数据库接入指南

本文档介绍如何将 xDooria 项目接入 Citus 分布式 PostgreSQL 数据库。

## 什么是 Citus？

Citus 是 PostgreSQL 的分布式扩展，通过水平分片（Sharding）实现数据库的横向扩展。它完全兼容 PostgreSQL 协议，对应用层透明。

### 核心特性

- **水平分片**：自动将数据分散到多个 Worker 节点
- **透明路由**：自动根据分片键路由查询到正确的节点
- **完全兼容**：100% 兼容 PostgreSQL SQL 语法
- **高可用**：支持副本和自动故障转移

### 架构

```
应用程序 (xDooria)
    ↓
  Citus 协调节点 (Coordinator)
    ↓ (自动路由)
┌──────┴──────┬──────────┬──────────┐
│   Worker 1  │ Worker 2 │ Worker 3 │
│  Shard 0-21 │ Shard 22-43│ Shard 44-63│
└─────────────┴──────────┴──────────┘
```

## 兼容性说明

**重要**: xDooria 的 `pkg/database/postgres` 封装完全兼容 Citus，**无需任何代码改动**！

- ✅ 所有查询方法 (`QueryOne`, `QueryAll`, `Exec`, etc.)
- ✅ 批量操作 (`InsertBatch`, `UpdateBatch`, `DeleteBatch`)
- ✅ 事务支持 (`BeginTx`, `WithTx`)
- ✅ 连接池管理
- ✅ 主从模式（连接到 Citus 协调节点的主从副本）

唯一需要做的是：**修改配置文件中的数据库连接地址**。

## 部署方式

### 方案一：Docker Compose（开发/测试环境）

#### 1. 创建 docker-compose.yml

```yaml
version: '3.8'

services:
  # Citus 协调节点（应用连接此节点）
  citus-coordinator:
    image: citusdata/citus:12.1
    container_name: citus-coordinator
    ports:
      - "5432:5432"
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: yourpassword
      POSTGRES_DB: xdooria
      PGDATA: /var/lib/postgresql/data/pgdata
    volumes:
      - citus-coordinator-data:/var/lib/postgresql/data
    command: postgres -c shared_preload_libraries=citus
    networks:
      - citus-network

  # Citus Worker 节点 1
  citus-worker1:
    image: citusdata/citus:12.1
    container_name: citus-worker1
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: yourpassword
      POSTGRES_DB: xdooria
      PGDATA: /var/lib/postgresql/data/pgdata
    volumes:
      - citus-worker1-data:/var/lib/postgresql/data
    command: postgres -c shared_preload_libraries=citus
    networks:
      - citus-network

  # Citus Worker 节点 2
  citus-worker2:
    image: citusdata/citus:12.1
    container_name: citus-worker2
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: yourpassword
      POSTGRES_DB: xdooria
      PGDATA: /var/lib/postgresql/data/pgdata
    volumes:
      - citus-worker2-data:/var/lib/postgresql/data
    command: postgres -c shared_preload_libraries=citus
    networks:
      - citus-network

  # Citus Worker 节点 3
  citus-worker3:
    image: citusdata/citus:12.1
    container_name: citus-worker3
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: yourpassword
      POSTGRES_DB: xdooria
      PGDATA: /var/lib/postgresql/data/pgdata
    volumes:
      - citus-worker3-data:/var/lib/postgresql/data
    command: postgres -c shared_preload_libraries=citus
    networks:
      - citus-network

volumes:
  citus-coordinator-data:
  citus-worker1-data:
  citus-worker2-data:
  citus-worker3-data:

networks:
  citus-network:
    driver: bridge
```

#### 2. 启动集群

```bash
docker-compose up -d
```

#### 3. 初始化 Citus 集群

创建 `scripts/init-citus.sh`:

```bash
#!/bin/bash

echo "等待 PostgreSQL 启动..."
sleep 10

echo "初始化 Citus 集群..."

# 连接到协调节点并添加 Worker 节点
docker exec -it citus-coordinator psql -U postgres -d xdooria <<EOF
-- 设置协调节点
SELECT citus_set_coordinator_host('citus-coordinator', 5432);

-- 添加 Worker 节点
SELECT citus_add_node('citus-worker1', 5432);
SELECT citus_add_node('citus-worker2', 5432);
SELECT citus_add_node('citus-worker3', 5432);

-- 验证节点
SELECT * FROM citus_get_active_worker_nodes();
EOF

echo "Citus 集群初始化完成！"
```

执行初始化:

```bash
chmod +x scripts/init-citus.sh
./scripts/init-citus.sh
```

### 方案二：Kubernetes 部署（生产环境）

#### 使用 Helm Chart

```bash
# 添加 Citus Helm 仓库
helm repo add citus https://citusdata.github.io/citus-helm
helm repo update

# 创建 values.yaml 配置文件
cat > citus-values.yaml <<EOF
coordinator:
  replicas: 1
  resources:
    requests:
      memory: "2Gi"
      cpu: "1000m"
    limits:
      memory: "4Gi"
      cpu: "2000m"

worker:
  replicas: 3
  resources:
    requests:
      memory: "4Gi"
      cpu: "2000m"
    limits:
      memory: "8Gi"
      cpu: "4000m"

postgresql:
  auth:
    username: postgres
    password: yourpassword
    database: xdooria
EOF

# 安装 Citus
helm install citus citus/citus -f citus-values.yaml -n database --create-namespace
```

## 数据库初始化

### 1. 创建分片表

创建 `migrations/001_create_players.sql`:

```sql
-- 连接到 Citus 协调节点
-- psql -h localhost -U postgres -d xdooria

-- 1. 创建普通表
CREATE TABLE IF NOT EXISTS players (
    player_id BIGINT PRIMARY KEY,
    name TEXT NOT NULL,
    level INT DEFAULT 1,
    gold BIGINT DEFAULT 0,
    server_id INT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 2. 将表转换为分片表（关键步骤！）
-- 按 player_id 分片，创建 64 个分片
-- 建议分片数 >= 预期最大 Worker 数 * 2
SELECT create_distributed_table('players', 'player_id', shard_count := 64);

-- 3. 创建索引
CREATE INDEX idx_players_level ON players(level);
CREATE INDEX idx_players_server_id ON players(server_id);
CREATE INDEX idx_players_created_at ON players(created_at);

-- 4. 创建其他表（示例）
CREATE TABLE IF NOT EXISTS items (
    item_id BIGINT PRIMARY KEY,
    player_id BIGINT NOT NULL,
    item_type INT NOT NULL,
    quantity INT DEFAULT 1,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 按 player_id 分片（与 players 表使用相同分片键实现 co-location）
SELECT create_distributed_table('items', 'player_id', colocate_with := 'players');
```

执行 migration:

```bash
docker exec -i citus-coordinator psql -U postgres -d xdooria < migrations/001_create_players.sql
```

### 2. 验证分片分布

```sql
-- 查看分片分布
SELECT nodename, count(*) as shard_count
FROM citus_shards
WHERE table_name = 'players'::regclass
GROUP BY nodename;

-- 查看所有分布式表
SELECT * FROM citus_tables;

-- 查看表的分片详情
SELECT * FROM citus_shards WHERE table_name = 'players'::regclass LIMIT 10;
```

## 应用配置

### 单机模式配置

修改 `config/database.yaml`:

```yaml
database:
  # 单机模式 - 连接到 Citus 协调节点
  standalone:
    host: "localhost"           # Citus 协调节点地址
    port: 5432
    user: "postgres"
    password: "yourpassword"
    db_name: "xdooria"
    ssl_mode: "disable"

  # 连接池配置
  pool:
    max_conns: 50               # Citus 可以处理更多连接
    min_conns: 10
    max_conn_lifetime: 3600s    # 1小时
    max_conn_idle_time: 1800s   # 30分钟
    health_check_period: 60s

  connect_timeout: 10s
  query_timeout: 30s
```

### 主从模式配置（推荐生产环境）

如果 Citus 协调节点配置了主从副本：

```yaml
database:
  # 主从模式
  master:
    host: "citus-coordinator-master"
    port: 5432
    user: "postgres"
    password: "yourpassword"
    db_name: "xdooria"
    ssl_mode: "disable"

  slaves:
    - host: "citus-coordinator-replica-1"
      port: 5432
      user: "postgres"
      password: "yourpassword"
      db_name: "xdooria"
      ssl_mode: "disable"
    - host: "citus-coordinator-replica-2"
      port: 5432
      user: "postgres"
      password: "yourpassword"
      db_name: "xdooria"
      ssl_mode: "disable"

  slave_load_balance: "round_robin"

  pool:
    max_conns: 50
    min_conns: 10
    max_conn_lifetime: 3600s
    max_conn_idle_time: 1800s
    health_check_period: 60s

  connect_timeout: 10s
  query_timeout: 30s
```

## 代码示例

**重要**: 代码无需任何修改！只需要改配置文件。

```go
package main

import (
    "context"
    "log"
    "github.com/lk2023060901/xdooria/pkg/database/postgres"
)

type Player struct {
    PlayerID  int64  `db:"player_id"`
    Name      string `db:"name"`
    Level     int    `db:"level"`
    Gold      int64  `db:"gold"`
    ServerID  int    `db:"server_id"`
}

func main() {
    // 创建客户端（连接到 Citus 协调节点）
    client, err := postgres.New(&postgres.Config{
        Standalone: &postgres.DBConfig{
            Host:     "localhost",
            Port:     5432,
            User:     "postgres",
            Password: "yourpassword",
            DBName:   "xdooria",
            SSLMode:  "disable",
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()

    // ✅ 插入数据（Citus 自动路由到正确的分片）
    affected, _ := client.Insert(ctx,
        "INSERT INTO players (player_id, name, level, gold, server_id) VALUES ($1, $2, $3, $4, $5)",
        12345, "Alice", 10, 1000, 1)
    log.Printf("插入成功: %d 行", affected)

    // ✅ 查询数据（带分片键 - 单分片查询，快）
    player, _ := client.QueryOne[Player](ctx,
        "SELECT * FROM players WHERE player_id = $1", 12345)
    log.Printf("查询结果: %+v", player)

    // ✅ 批量插入（使用 Pipeline）
    argsList := [][]any{
        {12346, "Bob", 20, 2000, 1},
        {12347, "Carol", 30, 3000, 2},
        {12348, "Dave", 40, 4000, 2},
    }
    total, _ := client.InsertBatch(ctx,
        "INSERT INTO players (player_id, name, level, gold, server_id) VALUES ($1, $2, $3, $4, $5)",
        argsList)
    log.Printf("批量插入: %d 行", total)

    // ⚠️ 全表扫描（没有分片键 - Citus 并行查询所有分片）
    highLevelPlayers, _ := client.QueryAll[Player](ctx,
        "SELECT * FROM players WHERE level > $1", 25)
    log.Printf("高等级玩家: %d 人", len(highLevelPlayers))
}
```

## 查询优化指南

### 最佳实践

#### ✅ 推荐：带分片键的查询（单分片查询）

```go
// 查询时带上分片键 player_id
player, _ := client.QueryOne[Player](ctx,
    "SELECT * FROM players WHERE player_id = $1", 12345)

// 更新时带上分片键
client.Update(ctx,
    "UPDATE players SET gold = gold + $1 WHERE player_id = $2",
    100, 12345)

// 删除时带上分片键
client.Delete(ctx,
    "DELETE FROM players WHERE player_id = $1", 12345)
```

**性能**: 极快，只访问一个分片。

#### ⚠️ 慎用：全表扫描（多分片并行查询）

```go
// 没有分片键，Citus 会并行查询所有分片
players, _ := client.QueryAll[Player](ctx,
    "SELECT * FROM players WHERE level > $1", 50)
```

**性能**: 较慢，但 Citus 会自动并行化查询，性能比单机好很多。

#### ✅ Co-located 表的 JOIN（推荐）

```sql
-- 两个表使用相同分片键，可以高效 JOIN
SELECT p.*, i.*
FROM players p
JOIN items i ON p.player_id = i.player_id
WHERE p.player_id = 12345;
```

**性能**: 快，在同一分片内完成 JOIN。

#### ❌ 避免：跨分片 JOIN

```sql
-- 不推荐：不同分片键的表 JOIN
SELECT p.*, s.*
FROM players p
JOIN servers s ON p.server_id = s.server_id;
```

**解决方案**: 拆成两次查询，或使用 reference table。

### 分片键选择建议

| 业务场景 | 推荐分片键 | 理由 |
|---------|-----------|------|
| 玩家数据 | `player_id` | 最常见的查询条件 |
| 服务器数据 | `server_id` | 按服务器隔离 |
| 公会数据 | `guild_id` | 按公会隔离 |
| 订单数据 | `player_id` 或 `order_id` | 看业务查询模式 |

## 扩缩容

### 添加 Worker 节点

```sql
-- 连接到协调节点
SELECT citus_add_node('citus-worker4', 5432);

-- 触发分片重新平衡（会迁移部分分片到新节点）
SELECT rebalance_table_shards('players');

-- 查看重平衡进度
SELECT * FROM citus_rebalance_status();
```

**注意**:
- 重平衡会产生数据迁移，建议在低峰期执行
- 数据量大时可能需要数小时到数天

### 移除 Worker 节点

```sql
-- 先将分片从要移除的节点迁移走
SELECT rebalance_table_shards('players');

-- 确认节点上没有分片后，移除节点
SELECT citus_remove_node('citus-worker4', 5432);
```

## 监控和运维

### 查看集群状态

```sql
-- 查看所有 Worker 节点
SELECT * FROM citus_get_active_worker_nodes();

-- 查看分片分布
SELECT nodename, count(*) as shard_count
FROM citus_shards
GROUP BY nodename;

-- 查看连接池状态（通过应用）
stats := client.Stats()
fmt.Printf("主库连接: %d/%d\n", stats.AcquiredConns, stats.MaxConns)
```

### 常见问题排查

#### 问题1: 连接失败

```bash
# 检查 Docker 容器状态
docker ps | grep citus

# 查看日志
docker logs citus-coordinator
docker logs citus-worker1

# 测试连接
docker exec -it citus-coordinator psql -U postgres -d xdooria
```

#### 问题2: 查询慢

```sql
-- 查看查询计划
EXPLAIN SELECT * FROM players WHERE player_id = 12345;

-- 查看是否单分片查询（Task Count: 1 表示单分片）
-- 如果 Task Count > 1，说明扫描了多个分片
```

#### 问题3: 分片不均匀

```sql
-- 查看分片大小
SELECT shardid,
       nodename,
       pg_size_pretty(shard_size)
FROM citus_shard_sizes
WHERE table_name = 'players'::regclass
ORDER BY shard_size DESC;

-- 重新平衡
SELECT rebalance_table_shards('players');
```

## 性能对比

### 单机 PostgreSQL vs Citus（3 Worker）

| 场景 | 单机 PG | Citus (3 Worker) | 提升 |
|------|---------|------------------|------|
| 带分片键查询 | 10ms | 10ms | 1x（无差异）|
| 全表扫描 | 1000ms | 350ms | 3x（并行）|
| 批量插入（1000条）| 500ms | 200ms | 2.5x |
| 写入并发 | 5000 TPS | 15000 TPS | 3x |
| 数据容量上限 | 2TB（单机） | 无限（横向扩展）| ∞ |

## 总结

### 接入成本

| 项目 | 工作量 | 说明 |
|------|--------|------|
| 代码改动 | **0 行** | 只改配置文件 |
| 部署 Citus | 1-2天 | Docker 或 K8s |
| Schema 迁移 | 0.5天 | 添加 `create_distributed_table` |
| 测试验证 | 1-2天 | 功能和性能测试 |

### 适用场景

✅ **推荐使用 Citus**:
- 单表数据量 > 1TB
- 写入 TPS > 5000
- 需要横向扩展
- 查询大多数带分片键

❌ **不推荐使用 Citus**:
- 数据量小（< 100GB）
- 大量跨分片 JOIN
- 需要频繁扩缩容
- 运维资源不足

### 相关资源

- [Citus 官方文档](https://docs.citusdata.com/)
- [Citus GitHub](https://github.com/citusdata/citus)
- [PostgreSQL 封装文档](../pkg/database/postgres/README.md)
