FROM golang:1.24-alpine AS builder

WORKDIR /build

# 复制 go mod 文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建
RUN CGO_ENABLED=0 GOOS=linux go build -o payment-service ./examples/jaeger/microservices/payment

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/payment-service .

EXPOSE 8083

CMD ["./payment-service"]
