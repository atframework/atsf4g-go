# Git Hooks 配置

本目录包含了项目的 Git hooks 脚本，用于自动化代码检查和格式化。

## 📋 包含的 Hooks

### pre-commit（提交前检查）
- **功能**：
  - 自动使用 `goimports` 格式化 Go 代码
  - 自动整理 import 语句
  - 使用 `golangci-lint` 检查代码质量
  - 按模块分别运行 lint（避免跨目录问题）
  - 智能判断是否需要阻止提交

- **文件**：
  - `pre-commit` - Bash 版本（Linux/Mac）
  - `pre-commit.ps1` - PowerShell 版本（Windows）
  - `pre-commit.bat` - Windows Batch 包装器

## 🚀 安装方式

### 🎯 自动安装（推荐 - 完全隔离）

本项目使用 Git 的 `core.hooksPath` 来隔离 hooks，确保仅在当前仓库生效，不影响其他仓库。

**Linux/Mac：**
```bash
bash script/hooks/install.sh
```

**Windows：**
```cmd
scripts\hooks\install.bat
```

安装脚本会自动配置 `core.hooksPath = script/hooks`，hooks 保存在仓库内。

### 🔒 工作原理

- ✅ 不修改全局 Git 配置（`~/.gitconfig`）
- ✅ 配置仅保存在 `.git/config`（单个仓库）
- ✅ 其他仓库完全不受影响
- ✅ Hooks 与代码一起版本控制

验证配置：
```bash
git config core.hooksPath
# 输出: script/hooks
```

## 📝 使用说明

安装后，当你执行 `git commit` 时，hook 会自动：

1. ✅ 检查提交的 Go 文件
2. ✅ 使用 `goimports` 自动格式化代码
3. ✅ 运行 `golangci-lint` 检查代码质量
4. ✅ 如果发现问题，阻止提交并显示错误信息

### 📋 查看执行日志

每次执行 pre-commit hook，日志会保存到：
```
script/hooks/pre-commit.log
```

**日志包含的信息：**
- ✅ 格式化的文件列表
- ✅ golangci-lint 的完整检查报告
- ✅ 任何错误或警告信息
- ✅ 执行时间戳

**查看最新的 hook 日志：**
```bash
cat script/hooks/pre-commit.log
```

每次执行都会覆盖之前的日志，保留最新的执行记录。

## 🔧 跳过 Hooks

如果需要跳过 hooks 检查（不推荐），可以使用 `--no-verify` 参数：

```bash
git commit --no-verify
```

## 📦 依赖要求

确保已安装以下工具：

- Go（golang）
- `goimports`：`go install golang.org/x/tools/cmd/goimports@latest`
- `golangci-lint`：`go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.6.0`

## 🔍 故障排除

### Hook 未执行
- 检查文件权限（Linux/Mac 需要执行权限）：`chmod +x .git/hooks/pre-commit`
- 确保 Git 配置中启用了 hooks

### golangci-lint 报错
- 检查 golangci-lint 是否安装：`golangci-lint --version`
- 查看 `.golangci.yaml` 配置文件
- 如果是路径问题，尝试使用 `git commit --no-verify` 并手动检查

### goimports 找不到
- 安装 goimports：`go install golang.org/x/tools/cmd/goimports@latest`
- 确保 `$GOPATH/bin` 在 `$PATH` 中
