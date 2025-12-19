# Redis Redlock 测试环境

本目录提供 Redlock 分布式锁测试所需的 3 个独立 Redis 实例。

## 环境说明

- **实例数量**: 3 个独立的 Redis 实例
- **端口**: 18001, 18002, 18003
- **容器名称**: xdooria-redis-redlock-1, xdooria-redis-redlock-2, xdooria-redis-redlock-3
- **内存限制**: 每个实例 256MB
- **持久化**: 禁用 (appendonly no, save "")

## 快速开始

### 启动环境

```bash
docker compose up -d
```

### 验证环境

```bash
# 检查所有容器状态
docker compose ps

# 测试连接
docker exec -it xdooria-redis-redlock-1 redis-cli ping
docker exec -it xdooria-redis-redlock-2 redis-cli ping
docker exec -it xdooria-redis-redlock-3 redis-cli ping
```

### 停止环境

```bash
docker compose down
```

## Redlock 算法说明

Redlock 是 Redis 官方推荐的分布式锁算法,用于在多个独立的 Redis 实例间实现可靠的分布式锁。

### 核心原理

1. **获取锁**: 按顺序向 N 个独立的 Redis 实例申请锁
2. **Quorum**: 需要在超过半数 (N/2 + 1) 的实例上成功获取锁
3. **时钟漂移**: 考虑时钟漂移因素,计算锁的有效时间
4. **释放锁**: 向所有实例发送释放锁命令

### 本环境配置

- **实例数**: 3
- **Quorum**: 2 (3/2 + 1 = 2)
- **推荐 TTL**: 5-10 秒

## 测试用例

运行 Redlock 测试:

```bash
cd ../../pkg/database/redis
go test -v -run TestRedlock
```

## 相关文档

- [Redis 分布式锁官方文档](https://redis.io/docs/manual/patterns/distributed-locks/)
- [Redlock 算法](https://redis.io/docs/reference/patterns/distributed-locks/#the-redlock-algorithm)
