#!/bin/bash
#
# 转发面板 编译脚本
# 构建后端、前端、节点客户端
#

set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
OUTPUT_DIR="$PROJECT_DIR/build"
rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"

echo "========================================"
echo "  转发面板 - 编译脚本"
echo "========================================"

# Build backend
echo ""
echo "[1/3] 编译后端..."
cd "$PROJECT_DIR/backend"
go mod download
go mod verify
CGO_ENABLED=1 go build -o "$OUTPUT_DIR/backend" ./cmd/
echo "  -> $OUTPUT_DIR/backend"

# Build node client
echo ""
echo "[2/3] 编译节点客户端..."
cd "$PROJECT_DIR/nodeclient"
go mod download
go mod verify
CGO_ENABLED=0 go build -o "$OUTPUT_DIR/nodeclient" .
echo "  -> $OUTPUT_DIR/nodeclient"

# Build frontend
echo ""
echo "[3/3] 编译前端..."
cd "$PROJECT_DIR/frontend"
npm ci --silent
npm run build
cp -r dist "$OUTPUT_DIR/public"
echo "  -> $OUTPUT_DIR/public/"

echo ""
echo "========================================"
echo "  编译完成!"
echo "  输出目录: $OUTPUT_DIR"
echo "========================================"
echo ""
echo "文件列表:"
ls -lh "$OUTPUT_DIR/"
