FROM golang:1.24-alpine AS builder

WORKDIR /build

# 复制 go mod 文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建
RUN CGO_ENABLED=0 GOOS=linux go build -o order-service ./examples/jaeger/microservices/order

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/order-service .

EXPOSE 8081

CMD ["./order-service"]
