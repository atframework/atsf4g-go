#!/bin/bash

# MkDocs 文档构建和测试脚本
# 用法: ./build.sh [serve|build|test]

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_info() {
    echo -e "${BLUE}ℹ ${1}${NC}"
}

print_success() {
    echo -e "${GREEN}✓ ${1}${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ ${1}${NC}"
}

print_error() {
    echo -e "${RED}✗ ${1}${NC}"
}

# 检查 Python 是否安装
check_python() {
    print_info "检查 Python 环境..."
    if ! command -v python3 &> /dev/null; then
        print_error "Python 3 未安装"
        exit 1
    fi
    print_success "Python 3 已安装: $(python3 --version)"
}

# 检查虚拟环境
check_venv() {
    if [ ! -d "venv" ]; then
        print_warning "虚拟环境不存在，正在创建..."
        python3 -m venv venv
        print_success "虚拟环境创建成功"
    fi
}

# 激活虚拟环境
activate_venv() {
    print_info "激活虚拟环境..."
    if [ -f "venv/bin/activate" ]; then
        source venv/bin/activate
        print_success "虚拟环境已激活"
    elif [ -f "venv/Scripts/activate" ]; then
        source venv/Scripts/activate
        print_success "虚拟环境已激活"
    else
        print_error "无法找到虚拟环境激活脚本"
        exit 1
    fi
}

# 安装依赖
install_deps() {
    print_info "安装 Python 依赖..."
    pip install --upgrade pip
    pip install -r requirements.txt
    print_success "依赖安装完成"
}

# 生成协议文档（.proto -> Markdown）
generate_proto_docs() {
    print_info "生成 Proto 协议文档..."
    if [ ! -f "scripts/generate_proto_docs.py" ]; then
        print_error "未找到 scripts/generate_proto_docs.py"
        exit 1
    fi
    python3 "scripts/generate_proto_docs.py"
    print_success "Proto 协议文档生成完成"
}

# 验证配置文件
validate_config() {
    print_info "验证 MkDocs 配置..."
    if [ ! -f "mkdocs.yml" ]; then
        print_error "mkdocs.yml 不存在"
        exit 1
    fi
    print_success "配置文件存在"
}

# 检查文档文件
check_docs() {
    print_info "检查文档文件..."
    
    local missing_files=0
    local required_files=(
        "docs/index.md"
        "docs/getting-started/index.md"
        "docs/architecture/overview.md"
        "docs/observability/index.md"
        "docs/faq.md"
        "docs/contributing.md"
    )
    
    for file in "${required_files[@]}"; do
        if [ ! -f "$file" ]; then
            print_warning "缺少文件: $file"
            ((missing_files++))
        fi
    done
    
    if [ $missing_files -eq 0 ]; then
        print_success "所有必需文件都存在"
    else
        print_warning "缺少 $missing_files 个文件"
    fi
}

# 构建文档
build_docs() {
    print_info "构建文档..."
    mkdocs build --clean --strict
    print_success "文档构建成功"
    
    # 显示构建结果
    if [ -d "site" ]; then
        local size=$(du -sh site | cut -f1)
        print_info "构建输出大小: $size"
    fi
}

# 启动开发服务器
serve_docs() {
    print_info "启动开发服务器..."
    print_info "访问 http://127.0.0.1:8000 查看文档"
    print_warning "按 Ctrl+C 停止服务器"
    mkdocs serve
}

# 测试文档
test_docs() {
    print_info "运行文档测试..."
    
    # 检查 Markdown 语法
    print_info "检查 Markdown 文件..."
    find docs -name "*.md" -type f | while read file; do
        if grep -q "^#[^#]" "$file"; then
            print_success "✓ $file"
        else
            print_warning "⚠ $file 可能缺少标题"
        fi
    done
    
    # 尝试构建
    print_info "尝试构建文档..."
    if mkdocs build --clean 2>&1 | tee build.log; then
        print_success "构建成功"
    else
        print_error "构建失败，请查看 build.log"
        exit 1
    fi
    
    # 检查死链接（如果安装了相关工具）
    if command -v linkchecker &> /dev/null; then
        print_info "检查链接..."
        linkchecker site/index.html
    else
        print_warning "linkchecker 未安装，跳过链接检查"
        print_info "安装: pip install linkchecker"
    fi
    
    print_success "所有测试通过"
}

# 清理构建产物
clean() {
    print_info "清理构建产物..."
    rm -rf site/
    rm -f build.log
    print_success "清理完成"
}

# 显示帮助信息
show_help() {
    cat << EOF
MkDocs 文档构建和测试脚本

用法: $0 [命令]

命令:
  serve     启动开发服务器（默认）
  build     构建静态网站
  test      运行文档测试
  clean     清理构建产物
  install   安装依赖
  help      显示此帮助信息

示例:
  $0 serve          # 启动开发服务器
  $0 build          # 构建文档
  $0 test           # 测试文档
  $0 clean          # 清理构建产物

EOF
}

# 主函数
main() {
    local command=${1:-serve}
    
    case $command in
        serve)
            check_python
            check_venv
            activate_venv
            install_deps
            validate_config
            generate_proto_docs
            serve_docs
            ;;
        build)
            check_python
            check_venv
            activate_venv
            install_deps
            validate_config
            generate_proto_docs
            build_docs
            ;;
        test)
            check_python
            check_venv
            activate_venv
            install_deps
            validate_config
            check_docs
            generate_proto_docs
            test_docs
            ;;
        clean)
            clean
            ;;
        install)
            check_python
            check_venv
            activate_venv
            install_deps
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            print_error "未知命令: $command"
            show_help
            exit 1
            ;;
    esac
}

# 运行主函数
main "$@"
