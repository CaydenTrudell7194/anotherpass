#!/bin/bash
#
# 转发面板 - 一键安装脚本
# 用法: bash <(curl -fsSL https://raw.githubusercontent.com/CaydenTrudell7194/anotherpass/main/deploy/install.sh)
#
set -e

REPO="CaydenTrudell7194/anotherpass"
VERSION="${1:-latest}"
INSTALL_DIR="/opt/forward-panel"
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  转发面板 v1.0 - 一键安装${NC}"
echo -e "${GREEN}========================================${NC}"

if [ "$EUID" -ne 0 ]; then
  echo -e "${RED}请以 root 用户运行${NC}"; exit 1
fi

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *) echo -e "${RED}不支持的架构: $(uname -m)${NC}"; exit 1 ;;
  esac
}

ARCH=$(detect_arch)

echo ""
echo -e "${YELLOW}[1/6] 安装 Docker...${NC}"
if ! command -v docker &> /dev/null; then
  curl -fsSL https://get.docker.com | sh
  systemctl enable docker; systemctl start docker
fi

if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null 2>&1; then
  DOCKER_COMPOSE_VERSION=$(curl -s https://api.github.com/repos/docker/compose/releases/latest | grep tag_name | cut -d'"' -f4)
  curl -L "https://github.com/docker/compose/releases/download/${DOCKER_COMPOSE_VERSION}/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
  chmod +x /usr/local/bin/docker-compose
fi

echo -e "${YELLOW}[2/6] 下载程序...${NC}"
mkdir -p "$INSTALL_DIR" && cd "$INSTALL_DIR"

if [ "$VERSION" = "latest" ]; then
  DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/forward-panel-linux-${ARCH}.tar.gz"
else
  DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/forward-panel-linux-${ARCH}.tar.gz"
fi

curl -fL "$DOWNLOAD_URL" -o panel.tar.gz || {
  echo -e "${RED}下载失败: $DOWNLOAD_URL${NC}"
  echo "请检查: 1.网络连接 2.仓库地址是否正确"
  exit 1
}
tar xzf panel.tar.gz && rm -f panel.tar.gz
chmod +x backend nodeclient 2>/dev/null || true

echo -e "${YELLOW}[3/6] 配置域名和密码...${NC}"
read -p "请输入面板域名 (如 panel.example.com): " DOMAIN
USE_HTTPS="y"
if [ -n "$DOMAIN" ] && ! [[ "$DOMAIN" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  read -p "启用 HTTPS (Let's Encrypt)? (Y/n) 选n则仅HTTP，可用于套CDN回源: " USE_HTTPS
  USE_HTTPS="${USE_HTTPS:-y}"
fi
read -p "设置管理员密码 (留空默认 admin123): " ADMIN_PWD
ADMIN_PWD="${ADMIN_PWD:-admin123}"

echo -e "${YELLOW}[4/6] 初始化数据库...${NC}"
MIGRATE=1 ADMIN="admin" ./backend 2>/dev/null || true

echo -e "${YELLOW}[5/6] 配置 Caddy 反向代理...${NC}"
mkdir -p caddy
if [ -z "$DOMAIN" ] || [[ "$DOMAIN" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  cat > caddy/Caddyfile << CADDYEOF
:80 {
  root * /opt/backend/public
  try_files {path} /index.html
  file_server
  reverse_proxy /api/* 127.0.0.1:18888
}
CADDYEOF
elif [ "$USE_HTTPS" = "n" ]; then
  cat > caddy/Caddyfile << CADDYEOF
${DOMAIN}:80 {
  root * /opt/backend/public
  try_files {path} /index.html
  file_server
  reverse_proxy /api/* 127.0.0.1:18888 {
    header_up X-Real-IP {http.request.header.CF-Connecting-IP}
  }
}
CADDYEOF
else
  cat > caddy/Caddyfile << CADDYEOF
${DOMAIN} {
  tls admin@${DOMAIN}
  root * /opt/backend/public
  try_files {path} /index.html
  file_server
  reverse_proxy /api/* 127.0.0.1:18888 {
    header_up X-Real-IP {http.request.header.CF-Connecting-IP}
  }
}
CADDYEOF
fi

cat > docker-compose.yml << DOCKEREOF
version: '3.8'
services:
  backend:
    image: debian:bookworm-slim
    network_mode: host
    restart: always
    volumes:
      - ${INSTALL_DIR}:/opt/backend
    working_dir: /opt/backend
    command: /opt/backend/backend
    environment:
      - LISTEN=127.0.0.1:18888
      - DATABASE=sqlite3:///opt/backend/data.db
    logging:
      driver: "json-file"
      options:
        max-size: "50m"  
        max-file: "3"
  caddy:
    image: caddy:2-alpine
    network_mode: host
    restart: always
    volumes:
      - ${INSTALL_DIR}:/opt/backend
      - ${INSTALL_DIR}/caddy:/etc/caddy
      - caddy_data:/data
    logging:
      driver: "json-file"
      options:
        max-size: "50m"
        max-file: "3"
volumes:
  caddy_data:
DOCKEREOF

echo -e "${YELLOW}[6/6] 启动服务...${NC}"
docker compose up -d

PROTO="https://"
ADDR="${DOMAIN}"
if [ -z "$DOMAIN" ] || [[ "$DOMAIN" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  PROTO="http://"
  ADDR="${DOMAIN:-$(curl -fsSL ifconfig.me)}"
elif [ "$USE_HTTPS" = "n" ]; then
  PROTO="http://"
fi
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  安装完成!${NC}"
echo -e "${GREEN}  面板地址: ${PROTO}${ADDR}${NC}"
echo -e "${GREEN}  管理员: admin${NC}"
echo -e "${GREEN}  密码: ${ADMIN_PWD}${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
NODE_PROTO="https://"
[ "$USE_HTTPS" = "n" ] || [ -z "$DOMAIN" ] || [[ "$DOMAIN" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]] && NODE_PROTO="http://"
echo "节点客户端安装命令 (在入口机运行):"
echo -e "  ${YELLOW}curl -fL https://github.com/${REPO}/releases/latest/download/nodeclient-linux-\$(uname -m).tar.gz -o /tmp/nc.tar.gz && tar xzf /tmp/nc.tar.gz -C /usr/local/bin/ && chmod +x /usr/local/bin/nodeclient${NC}"
echo ""
echo "对接入口机:"
echo -e "  ${YELLOW}nodeclient --server ${NODE_PROTO}${ADDR} --token <节点令牌> --device <设备组ID>${NC}"
