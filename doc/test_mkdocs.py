#!/usr/bin/env python3
"""
MkDocs 配置验证脚本
用于检查 mkdocs.yml 配置是否正确
"""

import sys
import yaml
from pathlib import Path


def print_success(msg):
    print(f"✓ {msg}")


def print_error(msg):
    print(f"✗ {msg}", file=sys.stderr)


def print_warning(msg):
    print(f"⚠ {msg}")


def check_mkdocs_config():
    """检查 mkdocs.yml 配置文件"""
    config_file = Path("mkdocs.yml")

    if not config_file.exists():
        print_error("mkdocs.yml 不存在")
        return False

    try:
        with open(config_file, 'r', encoding='utf-8') as f:
            config = yaml.safe_load(f)
        print_success("mkdocs.yml 格式正确")
    except yaml.YAMLError as e:
        print_error(f"mkdocs.yml 格式错误: {e}")
        return False

    # 检查必需的配置项
    required_keys = ['site_name', 'theme', 'nav']
    for key in required_keys:
        if key in config:
            print_success(f"配置项 '{key}' 存在")
        else:
            print_error(f"缺少必需的配置项 '{key}'")
            return False

    # 检查主题配置
    if 'name' in config.get('theme', {}):
        theme_name = config['theme']['name']
        print_success(f"主题: {theme_name}")
    else:
        print_error("主题配置不完整")
        return False

    return True


def check_docs_structure():
    """检查文档目录结构"""
    docs_dir = Path("docs")

    if not docs_dir.exists():
        print_error("docs 目录不存在")
        return False

    print_success("docs 目录存在")

    # 检查必需的文档文件
    required_files = [
        "docs/index.md",
        "docs/getting-started/index.md",
        "docs/architecture/index.md",
        "docs/observability/index.md",
    ]

    all_exist = True
    for file_path in required_files:
        if Path(file_path).exists():
            print_success(f"文件存在: {file_path}")
        else:
            print_warning(f"文件缺失: {file_path}")
            all_exist = False

    return all_exist


def check_assets():
    """检查资源文件"""
    assets_dir = Path("docs/assets")

    if not assets_dir.exists():
        print_warning("docs/assets 目录不存在，将在构建时创建")
        return True

    print_success("docs/assets 目录存在")

    # 列出资源文件
    asset_files = list(assets_dir.glob("*"))
    if asset_files:
        print_success(f"找到 {len(asset_files)} 个资源文件")
        for asset in asset_files:
            print(f"  - {asset.name}")
    else:
        print_warning("assets 目录为空")

    return True


def check_custom_files():
    """检查自定义 CSS 和 JS 文件"""
    custom_files = [
        "docs/stylesheets/extra.css",
        "docs/javascripts/mathjax.js",
    ]

    all_exist = True
    for file_path in custom_files:
        if Path(file_path).exists():
            print_success(f"自定义文件存在: {file_path}")
        else:
            print_warning(f"自定义文件缺失: {file_path}")
            all_exist = False

    return all_exist


def main():
    """主函数"""
    print("=" * 60)
    print("MkDocs 配置验证")
    print("=" * 60)
    print()

    results = []

    print("1. 检查 mkdocs.yml 配置")
    print("-" * 60)
    results.append(check_mkdocs_config())
    print()

    print("2. 检查文档目录结构")
    print("-" * 60)
    results.append(check_docs_structure())
    print()

    print("3. 检查资源文件")
    print("-" * 60)
    results.append(check_assets())
    print()

    print("4. 检查自定义文件")
    print("-" * 60)
    results.append(check_custom_files())
    print()

    print("=" * 60)
    if all(results):
        print_success("所有检查通过！")
        print()
        print("下一步:")
        print("  1. 安装依赖: pip install -r requirements.txt")
        print("  2. 启动服务: mkdocs serve")
        print("  3. 访问: http://127.0.0.1:8000")
        return 0
    else:
        print_error("部分检查未通过，请修复后重试")
        return 1


if __name__ == "__main__":
    sys.exit(main())
