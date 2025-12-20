FROM golang:1.25.5-alpine AS builder

WORKDIR /workspace

# 第一层：复制 go.mod 和 go.sum（根目录和示例）
COPY go.mod go.sum ./
COPY examples/observability/app/go.mod examples/observability/app/go.sum ./examples/observability/app/

# 第二层：下载依赖（只有 go.mod/go.sum 变化时才重新执行）
WORKDIR /workspace/examples/observability/app
ENV GOTOOLCHAIN=auto
RUN go mod download

# 第三层：复制源代码
WORKDIR /workspace
COPY . .

# 第四层：编译（只有源代码变化时才重新编译）
WORKDIR /workspace/examples/observability/app
RUN CGO_ENABLED=0 GOOS=linux go build -o app main.go

# 运行阶段
FROM alpine:latest

WORKDIR /app

# 复制编译好的二进制文件
COPY --from=builder /workspace/examples/observability/app/app .

# 暴露端口
EXPOSE 9090

# 运行应用
CMD ["./app"]
