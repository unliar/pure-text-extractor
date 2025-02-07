FROM golang:1.23-alpine AS builder
ENV GOPROXY=https://goproxy.cn,direct
# 多架构支持
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# 复制并下载依赖
COPY go.mod ./
COPY go.sum* ./
RUN go mod download

# 复制源代码
COPY . .

# 使用目标平台编译
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -a -installsuffix cgo -o main .

# 运行阶段
FROM alpine:latest

WORKDIR /app

# 复制可执行文件
COPY --from=builder /app/main .
# 运行时配置
EXPOSE 8080

# 根据操作系统选择正确的入口点
ENTRYPOINT ["./main"]
