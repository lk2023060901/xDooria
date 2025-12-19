# PostgreSQL Master-Slave 主从复制测试环境

主从复制环境，用于测试读写分离和主从同步功能。

## 架构说明

- **Master (主库)**: 处理写操作，端口 25432
- **Slave-1 (从库1)**: 只读副本，端口 25433
- **Slave-2 (从库2)**: 只读副本，端口 25434

## 配置说明

### Master (主库)
- **端口**: 25432
- **角色**: 读写
- **复制槽**: replication_slot_1, replication_slot_2

### Slave-1 (从库1)
- **端口**: 25433
- **角色**: 只读
- **复制模式**: 流复制 (Streaming Replication)

### Slave-2 (从库2)
- **端口**: 25434
- **角色**: 只读
- **复制模式**: 流复制 (Streaming Replication)

### 通用配置
- **用户**: xdooria
- **密码**: xdooria_pass
- **数据库**: xdooria_test
- **复制用户**: replicator
- **复制密码**: replicator_pass
- **版本**: PostgreSQL 16 Alpine

## 快速开始

### 启动环境

```bash
# 确保初始化脚本有执行权限
chmod +x init-master.sh init-slave.sh

# 启动所有服务
docker compose up -d

# 查看启动日志
docker compose logs -f
```

### 验证运行状态

```bash
# 检查所有容器状态
docker ps --filter name=xdooria-postgres

# 检查主库状态
docker exec xdooria-postgres-master psql -U xdooria -d xdooria_test -c "SELECT version();"

# 检查从库状态
docker exec xdooria-postgres-slave-1 psql -U xdooria -d xdooria_test -c "SELECT pg_is_in_recovery();"
docker exec xdooria-postgres-slave-2 psql -U xdooria -d xdooria_test -c "SELECT pg_is_in_recovery();"

# 查看复制状态（在主库上执行）
docker exec xdooria-postgres-master psql -U xdooria -d xdooria_test -c "SELECT * FROM pg_stat_replication;"
```

### 测试主从同步

```bash
# 在主库创建测试表
docker exec xdooria-postgres-master psql -U xdooria -d xdooria_test -c "
CREATE TABLE test_replication (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO test_replication (name) VALUES ('test1'), ('test2'), ('test3');
"

# 等待几秒让数据同步

# 在从库查询数据
docker exec xdooria-postgres-slave-1 psql -U xdooria -d xdooria_test -c "SELECT * FROM test_replication;"
docker exec xdooria-postgres-slave-2 psql -U xdooria -d xdooria_test -c "SELECT * FROM test_replication;"

# 清理测试表
docker exec xdooria-postgres-master psql -U xdooria -d xdooria_test -c "DROP TABLE test_replication;"
```

### 停止环境

```bash
docker compose down
```

### 清理数据

```bash
docker compose down -v
```

## 连接信息

在 Go 代码中使用以下配置连接：

```go
import "github.com/xDooria/xDooria/pkg/database/postgres"

config := &postgres.Config{
    Master: &postgres.DBConfig{
        Host:     "localhost",
        Port:     25432,
        User:     "xdooria",
        Password: "xdooria_pass",
        Database: "xdooria_test",
        SSLMode:  "disable",
    },
    Slaves: []postgres.DBConfig{
        {
            Host:     "localhost",
            Port:     25433,
            User:     "xdooria",
            Password: "xdooria_pass",
            Database: "xdooria_test",
            SSLMode:  "disable",
        },
        {
            Host:     "localhost",
            Port:     25434,
            User:     "xdooria",
            Password: "xdooria_pass",
            Database: "xdooria_test",
            SSLMode:  "disable",
        },
    },
    SlaveLoadBalance: "round_robin", // 或 "random"
    Pool: postgres.PoolConfig{
        MaxConns:        50,
        MinConns:        5,
        MaxConnLifetime: 1 * time.Hour,
        MaxConnIdleTime: 10 * time.Minute,
    },
}

client, err := postgres.New(config)
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

## 主从复制说明

### 复制延迟监控

```bash
# 在主库查看复制延迟
docker exec xdooria-postgres-master psql -U xdooria -d xdooria_test -c "
SELECT
    client_addr,
    state,
    sync_state,
    pg_wal_lsn_diff(pg_current_wal_lsn(), sent_lsn) AS sent_lag,
    pg_wal_lsn_diff(pg_current_wal_lsn(), write_lsn) AS write_lag,
    pg_wal_lsn_diff(pg_current_wal_lsn(), flush_lsn) AS flush_lag,
    pg_wal_lsn_diff(pg_current_wal_lsn(), replay_lsn) AS replay_lag
FROM pg_stat_replication;
"
```

### 从库只读验证

```bash
# 尝试在从库写入（应该失败）
docker exec xdooria-postgres-slave-1 psql -U xdooria -d xdooria_test -c "
CREATE TABLE should_fail (id INT);
"
# 预期错误: ERROR:  cannot execute CREATE TABLE in a read-only transaction
```

### 复制槽查看

```bash
# 查看复制槽状态
docker exec xdooria-postgres-master psql -U xdooria -d xdooria_test -c "
SELECT slot_name, slot_type, active, restart_lsn FROM pg_replication_slots;
"
```

## 故障排查

### 从库连接失败

```bash
# 检查从库日志
docker logs xdooria-postgres-slave-1
docker logs xdooria-postgres-slave-2

# 重启从库
docker compose restart postgres-slave-1
docker compose restart postgres-slave-2
```

### 复制停止

```bash
# 检查主库复制状态
docker exec xdooria-postgres-master psql -U xdooria -d xdooria_test -c "SELECT * FROM pg_stat_replication;"

# 查看复制槽
docker exec xdooria-postgres-master psql -U xdooria -d xdooria_test -c "SELECT * FROM pg_replication_slots;"

# 重建从库（如果复制完全中断）
docker compose down
docker compose up -d
```

### 端口冲突

修改 docker-compose.yaml 中的端口映射：

```yaml
ports:
  - "25435:5432"  # 改为其他端口
```

## 性能说明

为了提高测试速度，禁用了以下生产环境必需的特性：
- `fsync=off` - 禁用磁盘同步
- `synchronous_commit=off` - 异步提交
- `full_page_writes=off` - 禁用完整页写入
- 使用 tmpfs 内存文件系统

⚠️ **警告**: 这些设置仅用于测试环境，生产环境中会导致数据丢失！

## 相关文档

- [PostgreSQL 封装测试文档](../../../pkg/database/postgres/README_TEST.md)
- [PostgreSQL 复制文档](https://www.postgresql.org/docs/16/runtime-config-replication.html)
- [PostgreSQL 流复制](https://www.postgresql.org/docs/16/warm-standby.html)
