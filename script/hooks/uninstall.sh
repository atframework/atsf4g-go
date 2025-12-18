#!/bin/bash
# Uninstall git hooks for this repository
# This resets the core.hooksPath configuration and cleans up auto-generated hooks

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "🗑️  Uninstalling git hooks for this repository..."

# Check if we're in a git repository
if ! git -C "$REPO_ROOT" rev-parse --git-dir > /dev/null 2>&1; then
    echo "❌ Not a git repository: $REPO_ROOT"
    exit 1
fi

# 1. Reset core.hooksPath to default
echo "🔄 Resetting core.hooksPath..."
git -C "$REPO_ROOT" config --unset core.hooksPath || true
echo "✅ core.hooksPath reset"

# 2. Clean up auto-generated LFS hooks (these will be recreated by Git LFS if needed)
echo "🗑️  Removing auto-generated LFS hooks..."
for hook in post-checkout post-commit post-merge pre-push; do
    if [ -f "$SCRIPT_DIR/$hook" ]; then
        rm -f "$SCRIPT_DIR/$hook"
        echo "✅ Removed $hook"
    fi
done

# 3. Clean up log files
if [ -f "$SCRIPT_DIR/pre-commit.log" ]; then
    rm -f "$SCRIPT_DIR/pre-commit.log"
    echo "✅ Removed pre-commit.log"
fi

# 4. Reinstall Git LFS hooks to default location (.git/hooks/)
if command -v git-lfs &> /dev/null; then
    echo "🔄 Reinstalling Git LFS hooks to .git/hooks/..."
    git -C "$REPO_ROOT" lfs install --local --force 2>/dev/null || true
    echo "✅ Git LFS hooks reinstalled to default location"
fi

echo ""
echo "✅ Git hooks uninstalled successfully!"
echo "   • core.hooksPath has been reset"
echo "   • Auto-generated LFS hooks have been removed"
echo "   • Custom hooks remain in script/hooks/ (but are no longer active)"
echo ""
echo "To reinstall, run: bash script/hooks/install.sh"
