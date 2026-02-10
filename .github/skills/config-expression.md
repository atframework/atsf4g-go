# Configuration Loading & Expression Expansion

## 概述

配置加载系统支持从 YAML 文件和环境变量加载 protobuf 配置。
核心实现在 `atframework/libatapp-go/config.go`。

详细文档见子组件：`/atframework/libatapp-go/.github/skills/config-expression.md`

## 配置加载方式

### YAML 配置文件

```yaml
atapp:
  id: "0x00001234"
  bus:
    listen: "ipv6://:::21437"
    gateways:
      - address: "tcp://0.0.0.0:8080"
        match_labels:
          region: "us-east-1"
    receive_buffer_size: 8MB    # 支持 KB/MB/GB 单位
    ping_interval: 60s          # 支持 s/ms/m/h 单位
```

### 环境变量

```bash
ATAPP_ID=0x00001234
ATAPP_BUS_LISTEN=ipv6://:::21437
ATAPP_BUS_GATEWAYS_0_ADDRESS=tcp://0.0.0.0:8080
ATAPP_BUS_GATEWAYS_0_MATCH_LABELS_0_KEY=region
ATAPP_BUS_GATEWAYS_0_MATCH_LABELS_0_VALUE=us-east-1
```

### 加载优先级

环境变量（先）→ YAML（后，覆盖）→ Proto 默认值（最后填充）

## 表达式语法

对 `enable_expression: true` 的字段，支持以下表达式：

| 语法 | 说明 |
| --- | --- |
| `$VAR` | POSIX 标准变量 `[a-zA-Z_][a-zA-Z0-9_]*` |
| `${VAR}` | 花括号形式，支持 `.` `-` `/` 等 k8s label 字符 |
| `${VAR:-default}` | 变量未设置/为空时使用默认值 |
| `${VAR:+word}` | 变量已设置且非空时使用 word |
| `\$` | 转义为字面 `$` |
| `${OUTER_${INNER}}` | 嵌套变量名 |
| `${A:-${B:-default}}` | 多级嵌套默认值 |

### 使用示例

```yaml
atapp:
  bus:
    listen: "${BUS_LISTEN:-ipv6://:::21437}"
    gateways:
      - address: "${PROTO:-https}://${HOST}:${PORT:-443}"
        match_labels:
          "${LABEL_KEY:-env}": "${LABEL_VAL:-production}"
```

## 关键规则

1. 仅 `enable_expression: true` 的字段会展开表达式。
2. Map 的 key/value 通过父 map 字段的 `enable_expression` 控制。
3. 最大嵌套 32 层。
4. getter 可能返回 nil，总是做 nil-check。
5. 不要跨 RPC/reload 缓存 config struct 指针。

## 开发注意事项

- YAML 路径和环境变量路径是独立代码路径，新增功能两边都要改。
- `ParsePlainMessage` 不在 `LoadConfig` 流程中，不要只改它。
