# 使用官方Debian最新版本作为基础镜像
FROM debian:latest


ARG WORLD_ID="1"
ARG ZONE_ID="1"
# 更新包列表并安装必要的系统软件包
# 包括 curl（用于下载Helm）、gnupg（用于验证软件包）以及其他可能需要的工具
RUN apt-get update && apt-get install -y \
    curl \
    gnupg \
    ca-certificates \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# 安装 Helm（这里以安装 Helm 3.16.0 版本为例，您可以根据需要修改版本号）
RUN curl -fsSL curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

EXPOSE 7001

RUN mkdir -p /service

WORKDIR /service

COPY . /service

WORKDIR /service/lobbysvr/bin

CMD ["/service/lobbysvr/bin/lobbysvrd", "-config", "../cfg/lobbysvr.yaml", "-pid", "./lobbysvr.pid", "-crash-output-file", "../log/lobbysvr.crash.log", "start"]