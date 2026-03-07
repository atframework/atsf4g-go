# 可观测性

ATSF4G-GO 基于 atframework 提供的可观测性能力。

## 日志

### 日志配置

日志配置在 `atapp.yaml` 中：

```yaml
log:
  level: debug
  cat:
    number: 2
    prefix: log
  stacktrace:
    min: error
    max: fatal
```

### 日志级别

- `trace`: 最详细的调试信息
- `debug`: 调试信息
- `info`: 一般信息
- `notice`: 需要注意的信息
- `warn`: 警告信息
- `error`: 错误信息
- `fatal`: 致命错误

### 日志输出

日志输出到 `build/install/<服务名>/log/` 目录。

## 性能监控

### 内置指标

atframework 提供内置性能指标采集：

- 消息队列长度
- RPC 调用延迟
- 内存使用情况
- 协程/线程数量

### 自定义指标

在业务代码中可添加自定义指标采集。

## 调试

### 本地调试

使用 VS Code 调试配置：

```json
{
  "type": "go",
  "request": "launch",
  "name": "Launch lobbysvr",
  "program": "${workspaceFolder}/src/lobbysvr",
  "args": [],
  "cwd": "${workspaceFolder}/build/install/lobbysvr/bin"
}
```

### 远程调试

可以使用 delve 进行远程调试：

```bash
dlv --listen=:2345 --headless=true --api-version=2 exec ./lobbysvrd
```

## 日志分析

推荐使用以下工具分析日志：

- `grep`: 搜索特定日志
- `awk`: 统计和分析
- `jq`: 解析 JSON 格式日志（如果使用）
- VS Code 日志查看插件

## 故障排查

### 服务启动失败

1. 检查日志文件中的 error 和 fatal 级别日志
2. 确认配置文件格式正确
3. 检查端口是否被占用

### 性能问题

1. 查看日志中的性能指标
2. 使用 Go profiling 工具（pprof）
3. 检查数据库/缓存连接状态

### 内存泄漏

使用 Go 内置工具排查：

```bash
go tool pprof http://localhost:6060/debug/pprof/heap
```
