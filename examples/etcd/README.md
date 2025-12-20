# etcd 示例

本目录包含 xDooria etcd 封装的完整功能示例。

## 环境准备

### 启动 etcd 集群

```bash
cd /Volumes/work/code/golang/xDooria/xDooria/examples/etcd
docker-compose up -d
```

### 验证集群状态

```bash
docker exec xdooria-etcd1 etcdctl endpoint health
docker exec xdooria-etcd1 etcdctl member list
```

### 停止集群

```bash
docker-compose down -v
```

## 示例列表

### 1. basic - 基础使用

**功能点:**
- 创建客户端
- 健康检查
- 端点管理
- TLS 配置

**运行:**
```bash
cd basic
go run main.go
```

### 2. kv - 键值操作

**功能点:**
- Put/Get 基本操作
- 前缀查询
- 比较并交换 (CAS)
- PutIfNotExists
- 删除操作

**运行:**
```bash
cd kv
go run main.go
```

### 3. lease - 租约管理

**功能点:**
- 创建租约
- 自动续约 (KeepAlive)
- 获取 TTL
- 撤销租约

**运行:**
```bash
cd lease
go run main.go
```

### 4. watch - 监听功能

**功能点:**
- 监听单个键
- 监听前缀
- 停止监听
- 事件回调

**运行:**
```bash
cd watch
go run main.go
```

### 5. lock - 分布式锁

**功能点:**
- 基本锁操作
- 超时锁
- TryLock 非阻塞
- WithLockDo 辅助函数
- 并发锁竞争

**运行:**
```bash
cd lock
go run main.go
```

### 6. election - 选举功能

**功能点:**
- 基本选举
- 查询 Leader
- 观察 Leader 变化
- Leader 辞职
- 多节点竞选

**运行:**
```bash
cd election
go run main.go
```

### 7. transaction - 事务

**功能点:**
- 比较并交换 (CAS)
- 比较版本并更新
- 比较并删除
- 原子递增
- 复杂事务

**运行:**
```bash
cd transaction
go run main.go
```

## 集群信息

- **etcd1**: localhost:2379
- **etcd2**: localhost:22379
- **etcd3**: localhost:32379

## 注意事项

1. 确保 Docker 已安装并运行
2. 端口 2379, 22379, 32379 未被占用
3. 示例会自动连接到 localhost:2379
4. 运行示例前请先启动 etcd 集群
