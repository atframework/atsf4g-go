#!/bin/sh

# 编译脚本：将 C 文件编译成共享库

set -e

# 获取脚本目录（兼容 sh）
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
cd "$SCRIPT_DIR"

# 输出文件名
OUTPUT_LIB="libtest.so"

# 编译选项
CFLAGS="-fPIC -O2"
LDFLAGS="-shared"

echo "Compiling $OUTPUT_LIB..."

# 检测操作系统
case "$(uname -s)" in
    Linux*)
        MACHINE="Linux"
        ;;
    Darwin*)
        MACHINE="macOS"
        ;;
    *)
        MACHINE="Unknown"
        ;;
esac

# 编译
if [ "$MACHINE" = "Linux" ] || [ "$MACHINE" = "macOS" ]; then
    # Linux/macOS 编译
    gcc $CFLAGS -c libtest.c -o libtest.o
    gcc $LDFLAGS -o $OUTPUT_LIB libtest.o
    echo "✓ Successfully created $OUTPUT_LIB for $MACHINE"
else
    echo "Unsupported platform: $(uname -s)"
    exit 1
fi

# 清理中间文件
rm -f libtest.o

# 验证库文件
if [ -f "$OUTPUT_LIB" ]; then
    echo "✓ Library file created: $(pwd)/$OUTPUT_LIB"
    ls -lh $OUTPUT_LIB
else
    echo "✗ Failed to create library"
    exit 1
fi
