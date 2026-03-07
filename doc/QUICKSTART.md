# MkDocs 文档快速使用指南

## 一键启动（推荐）

### Windows (PowerShell)

```powershell
# 进入文档目录
cd doc

# 创建虚拟环境（首次使用）
python -m venv venv

# 激活虚拟环境
.\venv\Scripts\Activate.ps1

# 安装依赖（首次使用）
pip install -r requirements.txt

# 启动文档服务器
mkdocs serve

# 访问 http://127.0.0.1:8000
```

### Linux/macOS

```bash
# 进入文档目录
cd doc

# 创建虚拟环境（首次使用）
python3 -m venv venv

# 激活虚拟环境
source venv/bin/activate

# 安装依赖（首次使用）
pip install -r requirements.txt

# 启动文档服务器
mkdocs serve

# 访问 http://127.0.0.1:8000
```

## 常用命令

```bash
# 启动开发服务器（实时预览）
mkdocs serve

# 构建静态网站
mkdocs build

# 部署到 GitHub Pages
mkdocs gh-deploy

# 清理构建产物
rm -rf site/
```

## 文档编写

### 1. 创建新页面

在 `docs/` 目录下创建 Markdown 文件：

```bash
# 例如创建一个新的组件文档
touch docs/components/new-component.md
```

### 2. 添加到导航

编辑 `mkdocs.yml`，在 `nav` 部分添加：

```yaml
nav:
  - 核心组件:
      - components/index.md
      - 新组件: components/new-component.md  # 添加这一行
```

### 3. 编写内容

使用 Markdown 语法编写文档，支持：

- **代码高亮**: 使用 ` ```语言名 ` 包裹代码
- **Mermaid 图表**: 使用 ` ```mermaid ` 创建图表
- **数学公式**: 使用 `$公式$` 或 `$$公式$$`
- **告警框**: 使用 `!!! note` 等语法
- **选项卡**: 使用 `=== "标题"` 语法

### 4. 实时预览

运行 `mkdocs serve` 后，修改文档会自动刷新浏览器。

## Draw.io 图表集成

### 方法 1: 手动导出

1. 在 Draw.io 中打开 `architecture.drawio`
2. 文件 -> 导出为 -> SVG
3. 保存到 `docs/assets/architecture.svg`
4. 在 Markdown 中引用：
   ```markdown
   ![架构图](../assets/architecture.svg)
   ```

### 方法 2: 使用转换脚本（需要 Draw.io CLI）

```bash
# Linux/macOS
./scripts/convert_drawio.sh architecture.drawio docs/assets/architecture.svg

# Windows (需要先安装 Draw.io Desktop)
# 然后手动导出或使用 WSL 运行脚本
```

## Excalidraw 集成

### 方法 1: 导出为图片

1. 在 Excalidraw 中创建图表
2. 导出为 SVG 或 PNG
3. 保存到 `docs/assets/`
4. 在 Markdown 中引用

### 方法 2: 嵌入在线查看器

```markdown
[查看 Excalidraw 图表](https://excalidraw.com/#json=...)
```

## 常见问题

### Q: 如何修改主题颜色？

编辑 `mkdocs.yml`:

```yaml
theme:
  palette:
    - scheme: default
      primary: blue  # 修改为其他颜色
      accent: blue
```

### Q: 如何添加自定义 CSS？

编辑 `docs/stylesheets/extra.css`，添加您的样式。

### Q: Mermaid 图表不显示？

确保：
1. 使用正确的语法
2. 图表代码在 ` ```mermaid ` 代码块中
3. 浏览器支持 JavaScript

### Q: 如何优化构建速度？

1. 减少不必要的插件
2. 使用 `mkdocs serve --dirtyreload` 只重建修改的文件
3. 禁用 minify 插件（开发时）

## 部署

### GitHub Pages

```bash
# 自动部署
mkdocs gh-deploy

# 指定分支
mkdocs gh-deploy --remote-branch gh-pages
```

### 自定义服务器

```bash
# 构建静态文件
mkdocs build

# 将 site/ 目录部署到 Web 服务器
rsync -avz site/ user@server:/var/www/docs/
```

### Docker 部署

```dockerfile
FROM squidfunk/mkdocs-material:latest

WORKDIR /docs

COPY . .

RUN pip install -r requirements.txt

EXPOSE 8000

CMD ["mkdocs", "serve", "--dev-addr=0.0.0.0:8000"]
```

## 获取帮助

- 📖 MkDocs 官方文档: https://www.mkdocs.org
- 🎨 Material 主题文档: https://squidfunk.github.io/mkdocs-material
- 💬 项目 Issues: https://github.com/atframework/atsf4g-go/issues
