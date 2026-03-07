#!/bin/bash

# Wrapper kept for Unix users.
# The implementation lives in generate_proto_docs.py (cross-platform).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DOC_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

if command -v python3 >/dev/null 2>&1; then
  python3 "${SCRIPT_DIR}/generate_proto_docs.py" "$@"
elif command -v python >/dev/null 2>&1; then
  python "${SCRIPT_DIR}/generate_proto_docs.py" "$@"
else
  echo "[proto-docs] ERROR: python/python3 not found" 1>&2
  exit 1
fi
