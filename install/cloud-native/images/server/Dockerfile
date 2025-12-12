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

RUN mkdir -p /data/projecty

ARG SERVER_NAME="lobbysvr"

COPY . /data/projecty
COPY ./deploy/images/server/entrypoint.sh /entrypoint.sh
RUN sed -i 's/\r$//' /entrypoint.sh || true && \
	chmod +x /entrypoint.sh
# CMD ["/service/lobbysvr/bin/lobbysvrd", "-config", "../cfg/lobbysvr.yaml", "-pid", "./lobbysvr.pid", "-crash-output-file", "../log/lobbysvr.crash.log", "start"]

RUN set -eux; \
	if ! getent group tools >/dev/null 2>&1; then \
		groupadd -r tools || addgroup --system tools || true; \
	fi; \
	if ! id -u tools >/dev/null 2>&1; then \
		useradd -r -g tools -d /home/tools -s /usr/sbin/nologin -M tools || adduser --system --ingroup tools --no-create-home --shell /usr/sbin/nologin tools || true; \
	fi; \
	mkdir -p /home/tools /data/projecty; \
	chown -R tools:tools /home/tools /data/projecty || true

RUN chown -R tools:tools /data/projecty 

ENV SERVER_TYPE_NAME=${SERVER_NAME}

WORKDIR /data/projecty/${SERVER_NAME}/bin
USER tools

ENTRYPOINT [ "/bin/bash", "/entrypoint.sh" ]