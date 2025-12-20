FROM golang:1.23-alpine AS builder

WORKDIR /build

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖
ENV GOTOOLCHAIN=auto
RUN go mod download

# 复制源代码
COPY pkg ./pkg
COPY examples/prometheus/full_stack/app ./examples/prometheus/full_stack/app

# 构建应用
WORKDIR /build/examples/prometheus/full_stack/app
RUN CGO_ENABLED=0 GOOS=linux go build -o app main.go

# 运行阶段
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /build/examples/prometheus/full_stack/app/app .

EXPOSE 8080

CMD ["./app"]
