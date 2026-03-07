# Excalidraw 使用指南

## 概述

本文档使用 Excalidraw 来创建和展示架构图。Excalidraw 是一个开源的虚拟白板工具，可以创建手绘风格的图表。

## 架构图示例

![系统架构图](../assets/architecture-sample.excalidraw)

## 在线编辑

您可以使用以下方式编辑 Excalidraw 图表：

1. **在线编辑器**: 访问 [https://excalidraw.com](https://excalidraw.com)
2. **VS Code 插件**: 安装 Excalidraw 插件
3. **本地编辑**: 下载 `.excalidraw` 文件并在 Excalidraw 中打开

## 创建新图表

### 基本元素

Excalidraw 支持以下基本元素：

- **矩形**: 用于表示服务、组件
- **椭圆**: 用于表示数据库、存储
- **箭头**: 用于表示数据流、依赖关系
- **文本**: 用于添加标签和说明

### 样式设置

```json
{
  "strokeColor": "#1e1e1e",  // 边框颜色
  "backgroundColor": "#a5d8ff",  // 背景颜色
  "fillStyle": "solid",  // 填充样式: solid, hachure, cross-hatch
  "strokeWidth": 2,  // 边框宽度
  "strokeStyle": "solid",  // 边框样式: solid, dashed, dotted
  "roughness": 1,  // 手绘风格程度
  "opacity": 100  // 透明度
}
```

## 集成到 MkDocs

### 配置

本项目已在 `doc/mkdocs.yml` 中启用 Excalidraw 插件：

```yaml
plugins:
  - excalidraw
```

如需调整渲染参数，请参考插件文档并在 `mkdocs.yml` 中添加对应配置项。

### 在 Markdown 中使用

```markdown
![架构图](assets/architecture-sample.excalidraw)
```

## 最佳实践

### 1. 文件组织

```
docs/
├── assets/
│   ├── architecture-sample.excalidraw
│   ├── network.excalidraw
│   └── data-flow.excalidraw
└── architecture/
  └── index.md
```

### 2. 命名规范

- 使用描述性文件名：`system-architecture-sample.excalidraw`
- 使用小写字母和连字符：`data-flow.excalidraw`
- 避免空格和特殊字符

### 3. 图表设计

- **保持简洁**: 每个图表专注于一个主题
- **使用颜色编码**: 不同类型的组件使用不同颜色
- **添加图例**: 解释颜色和符号的含义
- **保持一致性**: 在所有图表中使用相同的样式

### 4. 版本控制

Excalidraw 文件是 JSON 格式，可以很好地进行版本控制：

```bash
# 查看图表变更
git diff docs/assets/architecture-sample.excalidraw

# 提交图表更新
git add docs/assets/*.excalidraw
git commit -m "更新架构图"
```
## 导出选项

Excalidraw 支持多种导出格式：

- **PNG**: 适合文档和演示
- **SVG**: 矢量格式，可缩放
- **JSON**: Excalidraw 原生格式
- **PDF**: 适合打印

## 协作功能

Excalidraw 支持实时协作：

1. 创建协作会话
2. 分享链接给团队成员
3. 实时编辑和讨论

## 快捷键

| 快捷键 | 功能 |
|--------|------|
| `R` | 矩形工具 |
| `D` | 菱形工具 |
| `O` | 椭圆工具 |
| `A` | 箭头工具 |
| `T` | 文本工具 |
| `Ctrl+Z` | 撤销 |
| `Ctrl+Y` | 重做 |
| `Ctrl+C` | 复制 |
| `Ctrl+V` | 粘贴 |

## 故障排除

### 图表不显示

1. 检查文件路径是否正确
2. 确认 Excalidraw 插件已安装
3. 检查文件格式是否正确

### 样式问题

1. 清除浏览器缓存
2. 检查 CSS 冲突
3. 更新 MkDocs 和插件版本

## 相关资源

- [Excalidraw 官网](https://excalidraw.com)
- [Excalidraw GitHub](https://github.com/excalidraw/excalidraw)
- [MkDocs Excalidraw 插件](https://github.com/ndy2/mkdocs-excalidraw)
