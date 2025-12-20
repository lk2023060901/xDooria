# gRPC Service Discovery Example

本示例演示如何使用 etcd 实现 gRPC 服务的自动注册与发现。

## 前置条件

确保已安装 Docker 和 Docker Compose。

## 运行步骤

### 1. 启动 etcd

```bash
cd examples/grpc/service_discovery
docker-compose up -d
```

验证 etcd 是否正常运行:

```bash
docker-compose ps
```

### 2. 启动 Server

```bash
go run server/main.go
```

Server 会自动注册到 etcd，服务名为 `xdooria.greeter`。

### 3. 启动 Client

在另一个终端运行:

```bash
go run client/main.go
```

Client 会从 etcd 自动发现 Server 地址并建立连接。

## 测试负载均衡

启动多个 Server 实例（使用不同端口）:

```bash
# Terminal 1
SERVER_PORT=50051 go run server/main.go

# Terminal 2
SERVER_PORT=50052 go run server/main.go

# Terminal 3
SERVER_PORT=50053 go run server/main.go
```

然后运行 Client，观察请求会被轮询分发到不同的 Server 实例。

## 清理

停止并删除 etcd 容器:

```bash
docker-compose down
```

## 关键特性

- **自动服务注册**: Server 启动时自动注册到 etcd
- **自动服务发现**: Client 通过 etcd resolver 自动发现服务
- **心跳保活**: Server 定期发送心跳，维持注册状态
- **优雅下线**: Server 停止时自动从 etcd 注销
- **负载均衡**: 支持 round_robin 等多种负载均衡策略
- **健康检查**: etcd 容器配置了健康检查
