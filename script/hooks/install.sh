#!/bin/bash
# Install git hooks for this repository only
# This uses Git's core.hooksPath to isolate hooks to this repo

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
HOOKS_DIR="$SCRIPT_DIR"

echo "📦 Installing git hooks for this repository only..."

# Check if we're in a git repository
if ! git -C "$REPO_ROOT" rev-parse --git-dir > /dev/null 2>&1; then
    echo "❌ Not a git repository: $REPO_ROOT"
    exit 1
fi

# Make hooks executable
for hook in pre-commit; do
    if [ -f "$HOOKS_DIR/$hook" ]; then
        chmod +x "$HOOKS_DIR/$hook"
        echo "✅ Marked $hook as executable"
    fi
done

# Set core.hooksPath to use hooks from scripts/hooks
# This is relative to the repository root
git -C "$REPO_ROOT" config core.hooksPath script/hooks

echo "✅ Git hooks installed successfully!"
