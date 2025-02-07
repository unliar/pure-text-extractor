#!/bin/bash

# 登录容器仓库
docker login host -u user -p password

# 定义仓库和版本
REGISTRY="host/unliar/simple-file-push"
VERSION=$(git rev-parse --short HEAD)

# 确保 Docker Buildx 可用
docker buildx create --use
docker buildx inspect --bootstrap

# 直接使用 buildx 构建并推送多架构镜像
docker buildx build \
    --platform linux/amd64,linux/arm64 \
    -t $REGISTRY:$VERSION \
    -t $REGISTRY:latest \
    --push \
    .