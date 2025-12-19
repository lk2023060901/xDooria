# PostgreSQL Standalone 测试环境

单机 PostgreSQL 测试环境，用于测试基本的数据库操作。

## 配置说明

- **端口**: 25432 (避免与本地 PostgreSQL 冲突)
- **用户**: xdooria
- **密码**: xdooria_pass
- **数据库**: xdooria_test
- **版本**: PostgreSQL 16 Alpine

## 性能优化（仅用于测试）

为了提高测试速度，禁用了以下生产环境必需的特性：
- `fsync=off` - 禁用磁盘同步
- `synchronous_commit=off` - 异步提交
- `full_page_writes=off` - 禁用完整页写入
- 使用 tmpfs 内存文件系统

⚠️ **警告**: 这些设置仅用于测试环境，生产环境中会导致数据丢失！

## 快速开始

### 启动环境

```bash
docker compose up -d
```

### 验证运行状态

```bash
# 检查容器状态
docker ps --filter name=xdooria-postgres-standalone

# 检查健康状态
docker exec xdooria-postgres-standalone pg_isready -U xdooria -d xdooria_test

# 连接到数据库
docker exec -it xdooria-postgres-standalone psql -U xdooria -d xdooria_test
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
    Standalone: &postgres.DBConfig{
        Host:     "localhost",
        Port:     25432,
        User:     "xdooria",
        Password: "xdooria_pass",
        Database: "xdooria_test",
        SSLMode:  "disable",
    },
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

## 查看日志

```bash
docker logs xdooria-postgres-standalone
```

## 常用命令

```bash
# 进入 psql 交互式终端
docker exec -it xdooria-postgres-standalone psql -U xdooria -d xdooria_test

# 列出所有表
docker exec -it xdooria-postgres-standalone psql -U xdooria -d xdooria_test -c "\dt"

# 查看数据库大小
docker exec -it xdooria-postgres-standalone psql -U xdooria -d xdooria_test -c "SELECT pg_size_pretty(pg_database_size('xdooria_test'));"

# 查看连接数
docker exec -it xdooria-postgres-standalone psql -U xdooria -d xdooria_test -c "SELECT count(*) FROM pg_stat_activity;"
```

## 故障排查

### 端口冲突

如果端口 25432 已被占用：

```bash
# 检查端口占用
lsof -i :25432

# 修改 docker-compose.yaml 中的端口映射
ports:
  - "25433:5432"  # 改为其他端口
```

### 连接失败

```bash
# 检查容器是否运行
docker ps --filter name=xdooria-postgres-standalone

# 查看容器日志
docker logs xdooria-postgres-standalone

# 重启容器
docker compose restart
```

## 相关文档

- [PostgreSQL 封装测试文档](../../../pkg/database/postgres/README_TEST.md)
- [PostgreSQL 官方文档](https://www.postgresql.org/docs/16/)
