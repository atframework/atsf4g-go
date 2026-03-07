# Draw.io 图表使用指南

本文档介绍如何在 MkDocs 中使用 Draw.io 图表。

## 什么是 Draw.io？

Draw.io（现在也称为 diagrams.net）是一个功能强大的免费图表绘制工具，支持：

- 流程图
- UML 图
- 网络拓扑图
- 架构图
- ER 图
- 思维导图

## 在 MkDocs 中使用 Draw.io

### 方法 1: 使用 mkdocs-drawio 插件（推荐）

#### 1. 插件已配置

插件已在 `doc/mkdocs.yml` 中启用：

```yaml
plugins:
  - drawio
```

如需自定义转换行为，可扩展为：

```yaml
plugins:
  - drawio:
      sources: "*.drawio"
      drawio_executable: null
      format: svg
      embed_format: "{img_open}{img_src}{img_close}"
```

#### 2. 直接使用 .drawio 文件

将 `.drawio` 文件放到文档目录，然后在 Markdown 中引用：

```markdown
# 系统架构

![架构图](architecture.drawio)
```

插件会自动将 `.drawio` 文件转换为 SVG 并嵌入到页面中。

#### 3. 配置选项说明

| 选项 | 说明 | 默认值 |
|------|------|--------|
| `sources` | 匹配的文件模式 | `"*.drawio"` |
| `drawio_executable` | Draw.io CLI 路径 | `null`（自动检测） |
| `format` | 输出格式 | `svg` |
| `embed_format` | 嵌入格式 | 标准 img 标签 |

### 方法 2: 使用现有的 architecture.drawio

项目中已提供示例文件：`doc/docs/assets/architecture.drawio`。

通常不需要额外复制，直接在 Markdown 中引用即可。

#### 选项 A: 直接引用（推荐）

在 `doc/docs/architecture/index.md`（或任何其他页面）中：

```markdown
![整体架构](../assets/architecture.drawio)
```

#### 选项 B: 导出为 SVG（可选）

```bash
# 使用 Draw.io Desktop 导出
# 文件 -> 导出为 -> SVG

# 或使用命令行（需要安装 Draw.io Desktop）
drawio -x -f svg -o docs/assets/architecture.svg architecture.drawio
```

### 方法 3: 在线编辑

1. 访问 https://app.diagrams.net/
2. 打开现有的 `.drawio` 文件
3. 编辑后保存
4. 将文件放到 `docs/assets/` 目录

## 使用示例

### 基本用法

假设您有一个 `docs/assets/system-flow.drawio` 文件：

```markdown
## 系统流程

![系统流程图](assets/system-flow.drawio)

上图展示了系统的主要流程。
```

### 多页面图表

Draw.io 支持多页面图表，插件会自动处理：

```markdown
## 架构图

### 整体架构
![整体架构](assets/architecture.drawio#page-0)

### 详细设计
![详细设计](assets/architecture.drawio#page-1)
```

### 指定尺寸

```markdown
<img src="assets/architecture.drawio" alt="架构图" width="800">
```

## 转换现有的 architecture.drawio

项目中的 `doc/architecture.drawio` 包含两个页面：

1. **整体架构-服务关系**
2. **其他页面**

### 步骤 1: 复制文件

如果你要新增自己的图表文件，请将其放入 `doc/docs/assets/`，例如：

- `doc/docs/assets/my-architecture.drawio`

### 步骤 2: 在文档中使用

在 `doc/docs/architecture/index.md`（或其他页面）中添加：

```markdown
## 原始架构图（Draw.io）

### 整体架构-服务关系

![整体架构](../assets/architecture.drawio#page-0)

这是使用 Draw.io 绘制的原始架构图。

### 页面 2

![详细设计](../assets/architecture.drawio#page-1)
```

### 步骤 3: 构建文档

```bash
mkdocs build
```

插件会自动将 `.drawio` 文件转换为 SVG。

## Draw.io 插件 vs 手动导出

### Draw.io 插件优势

- ✅ 自动转换，无需手动导出
- ✅ 保留源文件，易于编辑
- ✅ 支持多页面图表
- ✅ 构建时自动更新

### 手动导出优势

- ✅ 不依赖插件
- ✅ 更快的构建速度
- ✅ 更好的浏览器兼容性
- ✅ 可以手动优化 SVG

### 推荐方案

**开发阶段**：使用 Draw.io 插件，方便快速迭代

**生产部署**：导出为 SVG，获得更好的性能

## 配置 Draw.io Desktop（可选）

如果需要使用命令行转换，需要安装 Draw.io Desktop：

### Windows

```powershell
# 下载并安装 Draw.io Desktop
# https://github.com/jgraph/drawio-desktop/releases

# 添加到 PATH
$env:PATH += ";C:\Program Files\draw.io"

# 测试
drawio --version
```

### macOS

```bash
# 使用 Homebrew 安装
brew install --cask drawio

# 测试
/Applications/draw.io.app/Contents/MacOS/draw.io --version
```

### Linux

```bash
# 下载 .deb 或 .rpm 包
wget https://github.com/jgraph/drawio-desktop/releases/download/v22.1.2/drawio-amd64-22.1.2.deb

# 安装
sudo dpkg -i drawio-amd64-22.1.2.deb

# 测试
drawio --version
```

## 批量转换脚本

如果有多个 `.drawio` 文件需要转换：

```bash
#!/bin/bash
# convert_all_drawio.sh

for file in docs/**/*.drawio; do
    output="${file%.drawio}.svg"
    echo "Converting $file to $output"
    drawio -x -f svg -o "$output" "$file"
done
```

## 最佳实践

### 1. 文件组织

```
docs/
├── assets/
│   ├── diagrams/
│   │   ├── architecture.drawio
│   │   ├── network.drawio
│   │   └── data-flow.drawio
│   └── images/
│       ├── screenshot1.png
│       └── screenshot2.png
```

### 2. 命名规范

- 使用小写字母和连字符
- 使用描述性名称
- 例如：`user-authentication-flow.drawio`

### 3. 版本控制

```bash
# 同时提交源文件和导出文件
git add docs/assets/architecture.drawio
git add docs/assets/architecture.svg
git commit -m "Update architecture diagram"
```

### 4. 文档说明

在使用图表的地方添加说明：

```markdown
## 系统架构

![系统架构](assets/architecture.drawio)

**图表说明**：
- 蓝色框：客户端组件
- 橙色框：网关服务
- 绿色框：业务服务
- 红色框：数据存储

**编辑方法**：
1. 使用 Draw.io Desktop 打开 `docs/assets/architecture.drawio`
2. 编辑后保存
3. 重新构建文档：`mkdocs build`
```

## 图表类型示例

### 流程图

```markdown
![用户登录流程](assets/login-flow.drawio)
```

### UML 类图

```markdown
![系统类图](assets/class-diagram.drawio)
```

### 网络拓扑图

```markdown
![网络拓扑](assets/network-topology.drawio)
```

### ER 图

```markdown
![数据库 ER 图](assets/database-er.drawio)
```

## 常见问题

### Q: Draw.io 文件不显示？

**A:** 检查：
1. 文件路径是否正确
2. 插件是否正确配置
3. Draw.io Desktop 是否已安装（如果使用 CLI）

### Q: 如何编辑已有的 .drawio 文件？

**A:** 
1. 使用 Draw.io Desktop 打开
2. 或访问 https://app.diagrams.net/ 在线编辑
3. 编辑后保存并重新构建文档

### Q: 转换速度慢怎么办？

**A:** 
1. 预先导出为 SVG
2. 使用缓存
3. 只在需要时重新转换

### Q: 支持哪些输出格式？

**A:** 
- SVG（推荐）
- PNG
- PDF
- JPEG

配置示例：
```yaml
plugins:
  - drawio:
      format: png  # 或 svg, pdf, jpeg
```

### Q: 如何优化 SVG 文件大小？

**A:** 
```bash
# 使用 SVGO 优化
npm install -g svgo
svgo architecture.svg -o architecture.optimized.svg
```

## Draw.io vs Excalidraw vs Mermaid

| 特性 | Draw.io | Excalidraw | Mermaid |
|------|---------|------------|---------|
| 直接渲染 | ✅ | ✅ | ✅ |
| 功能丰富度 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ |
| 易用性 | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| 手绘风格 | ❌ | ✅ | ❌ |
| 代码定义 | ❌ | ❌ | ✅ |
| 专业图表 | ✅ | ❌ | ⚠️ |
| 学习曲线 | 中 | 低 | 低 |

### 选择建议

| 场景 | 推荐工具 |
|------|---------|
| 复杂的技术架构图 | Draw.io |
| 快速草图和概念图 | Excalidraw |
| 代码化的流程图 | Mermaid |
| UML 图 | Draw.io |
| 网络拓扑图 | Draw.io |
| 演示用图表 | Excalidraw |

## 相关资源

- [Draw.io 官网](https://www.diagrams.net/)
- [Draw.io GitHub](https://github.com/jgraph/drawio)
- [Draw.io Desktop](https://github.com/jgraph/drawio-desktop)
- [mkdocs-drawio 插件](https://pypi.org/project/mkdocs-drawio/)
- [Draw.io 模板库](https://www.diagrams.net/blog/template-diagrams)

## 下一步

- 尝试将 `doc/architecture.drawio` 集成到文档中
- 创建新的 Draw.io 图表
- 探索 Draw.io 的高级功能
- 结合 Mermaid 和 Excalidraw 使用
