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
echo "  转发面板 - 编译脚本 (授权版)"
echo "========================================"

# 检查 garble
HAS_GARBLE=false
if command -v garble &> /dev/null; then
  HAS_GARBLE=true
  echo "  garble: 已安装 ✅"
else
  echo "  garble: 未安装，将使用标准 go build (开发模式)"
fi

# Build backend
echo ""
echo "[1/3] 编译后端..."
cd "$PROJECT_DIR/backend"
go mod download
go mod verify

BUILD_TIME=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
GIT_HASH=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS="-s -w -X 'forward-panel/version.BuildTime=${BUILD_TIME}' -X 'forward-panel/version.GitHash=${GIT_HASH}'"

if [ "$HAS_GARBLE" = true ]; then
  echo "  使用 garble 混淆编译..."
  GARBLE_SEED=$(openssl rand -hex 16 2>/dev/null || head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n')
  
  # 处理授权公钥：通过环境变量注入，不在源码中硬编码
  if [ -n "${LICENSE_PUBLIC_KEY:-}" ]; then
    LDFLAGS="${LDFLAGS} -X 'forward-panel/license.publicKeyPEM=${LICENSE_PUBLIC_KEY}'"
  fi
  
  # 注意：crypto.c 依赖 openssl，需要安装 libssl-dev
  CGO_ENABLED=1 \
    CC="${CC:-gcc}" \
    garble -tiny -literals -split-shapes -seed="${GARBLE_SEED}" \
    go build -ldflags="${LDFLAGS}" -buildvcs=false -o "$OUTPUT_DIR/backend" ./cmd/
else
  echo "  使用标准 go build (开发模式)..."
  CGO_ENABLED=1 go build -ldflags="${LDFLAGS}" -buildvcs=false -o "$OUTPUT_DIR/backend" ./cmd/
fi
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
