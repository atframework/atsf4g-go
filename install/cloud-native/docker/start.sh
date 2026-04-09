#!/bin/bash
set -euo pipefail

# 云下 Docker 启动脚本
# 由 docker-compose.yml 调用，在容器内运行
# 流程: atdtool template 生成配置 -> 切换到 tools 用户 -> 启动 lobbysvrd

PROJECT_ROOT_DIR="/data/projecty"
SERVER_TYPE_NAME="${SERVER_TYPE_NAME:-lobbysvr}"
WORLD_ID="${WORLD_ID:-1}"
ZONE_ID="${ZONE_ID:-1}"
DEPLOY_ENV="${DEPLOY_ENV:-production}"
EXTRA_SET_VALUES="${EXTRA_SET_VALUES:-}"

WORKDIR="${PROJECT_ROOT_DIR}/${SERVER_TYPE_NAME}/bin"

echo "$(date '+%Y/%m/%d %H:%M:%S') [INFO] === lobbysvr non-k8s docker startup ==="
echo "$(date '+%Y/%m/%d %H:%M:%S') [INFO] WORLD_ID=${WORLD_ID} ZONE_ID=${ZONE_ID} DEPLOY_ENV=${DEPLOY_ENV}"

# ---- Step 1: 用 atdtool template 生成配置 ----
echo "$(date '+%Y/%m/%d %H:%M:%S') [INFO] generating configuration via atdtool template ..."

ATDTOOL="${PROJECT_ROOT_DIR}/atdtool/atdtool"
CHARTS_DIR="${PROJECT_ROOT_DIR}/deploy/charts"
VALUES_DEFAULT="${PROJECT_ROOT_DIR}/deploy/values/default"
VALUES_ENV="${PROJECT_ROOT_DIR}/deploy/values/${DEPLOY_ENV}"

chmod +x "${ATDTOOL}"

# 构建 --values 参数
VALUES_PATHS="${VALUES_DEFAULT}"
if [[ -d "${VALUES_ENV}" ]]; then
  VALUES_PATHS="${VALUES_PATHS},${VALUES_ENV}"
fi

# 构建 --set 参数
SET_ARGS=(--set "global.world_id=${WORLD_ID}" --set "global.zone_id=${ZONE_ID}")
if [[ -n "${EXTRA_SET_VALUES}" ]]; then
  SET_ARGS+=(--set "${EXTRA_SET_VALUES}")
fi

"${ATDTOOL}" template "${CHARTS_DIR}" \
  -o "${PROJECT_ROOT_DIR}" \
  --values "${VALUES_PATHS}" \
  "${SET_ARGS[@]}"

echo "$(date '+%Y/%m/%d %H:%M:%S') [INFO] configuration generated"

# ---- Step 2: 找到生成的配置文件 ----
CFG_DIR="${PROJECT_ROOT_DIR}/${SERVER_TYPE_NAME}/cfg"
CONFIG_FILE=$(find "${CFG_DIR}" -maxdepth 1 -name "${SERVER_TYPE_NAME}_*.yaml" ! -name "vector_*" | head -1)

if [[ -z "${CONFIG_FILE}" ]]; then
  echo "$(date '+%Y/%m/%d %H:%M:%S') [ERROR] no config file found in ${CFG_DIR}"
  exit 1
fi

echo "$(date '+%Y/%m/%d %H:%M:%S') [INFO] using config: ${CONFIG_FILE}"

# 从配置文件名提取 bus_addr (如 lobbysvr_1.1.11.1.yaml -> 1.1.11.1)
BUS_ADDR=$(basename "${CONFIG_FILE}" .yaml | sed "s/${SERVER_TYPE_NAME}_//")

# ---- Step 4: 修复目录权限，切换到 tools 用户启动 ----
LOG_DIR="${PROJECT_ROOT_DIR}/${SERVER_TYPE_NAME}/log"
mkdir -p "${LOG_DIR}"
chown -R tools:tools "${PROJECT_ROOT_DIR}/${SERVER_TYPE_NAME}"

PID_FILE="${WORKDIR}/${SERVER_TYPE_NAME}.pid"
CRASH_LOG="${LOG_DIR}/${SERVER_TYPE_NAME}_${BUS_ADDR}.crash.log"

echo "$(date '+%Y/%m/%d %H:%M:%S') [INFO] starting ${SERVER_TYPE_NAME}d (bus_addr=${BUS_ADDR}) ..."

# 前台运行 lobbysvrd（不加 &），让 Docker 能正确管理进程生命周期
cd "${WORKDIR}"
exec su -s /bin/bash tools -c \
  "exec ${WORKDIR}/${SERVER_TYPE_NAME}d \
    -pid '${PID_FILE}' \
    -config '${CONFIG_FILE}' \
    -crash-output-file '${CRASH_LOG}' \
    start"
