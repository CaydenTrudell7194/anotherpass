#!/bin/bash
#
# 转发面板 - 一键安装脚本
# 用法: bash <(curl -fsSL https://raw.githubusercontent.com/CaydenTrudell7194/anotherpass/main/deploy/install.sh)
#
set -euo pipefail

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

is_ip() {
	[[ "$1" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]] || [[ "$1" =~ ^\[?[0-9a-fA-F:]+\]?$ ]]
}

is_domain() {
	[[ "$1" =~ ^([A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?\.)+[A-Za-z]{2,63}$ ]]
}

echo ""
echo -e "${YELLOW}[1/6] 安装 Docker...${NC}"
if ! command -v docker &> /dev/null; then
  curl -fsSL https://get.docker.com | sh
  systemctl enable docker; systemctl start docker
fi

if docker compose version &> /dev/null; then
  COMPOSE=(docker compose)
elif command -v docker-compose &> /dev/null; then
  COMPOSE=(docker-compose)
else
  DOCKER_COMPOSE_VERSION=$(curl -s https://api.github.com/repos/docker/compose/releases/latest | grep tag_name | cut -d'"' -f4)
  curl -fL "https://github.com/docker/compose/releases/download/${DOCKER_COMPOSE_VERSION}/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
  chmod +x /usr/local/bin/docker-compose
  COMPOSE=(docker-compose)
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
if [ -n "$DOMAIN" ] && ! is_ip "$DOMAIN" && ! is_domain "$DOMAIN"; then
  echo -e "${RED}域名格式无效${NC}"
  exit 1
fi
USE_HTTPS="y"
if [ -n "$DOMAIN" ] && ! is_ip "$DOMAIN"; then
  read -p "启用 HTTPS (Let's Encrypt)? (Y/n) 选n则仅HTTP，可用于套CDN回源: " USE_HTTPS
  USE_HTTPS="${USE_HTTPS:-y}"
fi
umask 077
LEGACY_UPGRADE="n"
if [ -f data.db ] && [ ! -f panel.env ]; then
  LEGACY_UPGRADE="y"
fi
if [ -f panel.env ]; then
  echo "检测到已有配置，将保留管理员密码和 JWT 密钥"
  ADMIN_PWD=""
elif [ "$LEGACY_UPGRADE" = "y" ]; then
  echo "检测到旧版数据库，将保留原管理员密码并生成新的 JWT 密钥"
  ADMIN_PWD=""
  BOOTSTRAP_PWD=$(openssl rand -hex 16 2>/dev/null || head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n')
  JWT_SECRET=$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n')
  cat > panel.env << ENVEOF
ADMIN_PASSWORD=${BOOTSTRAP_PWD}
JWT_SECRET=${JWT_SECRET}
DATABASE=sqlite3:///data/data.db
ENVEOF
else
  read -s -p "设置管理员密码 (至少8位，仅支持字母、数字和常用符号): " ADMIN_PWD
  echo ""
  if [ ${#ADMIN_PWD} -lt 8 ] || ! [[ "$ADMIN_PWD" =~ ^[A-Za-z0-9@%_+=:,\.\!\?-]+$ ]]; then
    echo -e "${RED}密码至少8位，且不能包含空格、引号、$或换行${NC}"
    exit 1
  fi
  JWT_SECRET=$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n')
  cat > panel.env << ENVEOF
ADMIN_PASSWORD=${ADMIN_PWD}
JWT_SECRET=${JWT_SECRET}
DATABASE=sqlite3:///data/data.db
ENVEOF
fi
mkdir -p data caddy
chmod 700 data
chmod 600 panel.env
if [ -f data.db ] && [ ! -f data/data.db ]; then
  mv data.db data/data.db
fi

echo -e "${YELLOW}[4/6] 配置 Caddy 反向代理...${NC}"
if [ -z "$DOMAIN" ] || is_ip "$DOMAIN"; then
  cat > caddy/Caddyfile << CADDYEOF
:80 {
  reverse_proxy /api/* 127.0.0.1:18888
  handle {
    root * /srv/public
    try_files {path} /index.html
    file_server
  }
}
CADDYEOF
elif [ "$USE_HTTPS" = "n" ]; then
  cat > caddy/Caddyfile << CADDYEOF
${DOMAIN}:80 {
  reverse_proxy /api/* 127.0.0.1:18888 {
    header_up X-Real-IP {http.request.header.CF-Connecting-IP}
  }
  handle {
    root * /srv/public
    try_files {path} /index.html
    file_server
  }
}
CADDYEOF
else
  cat > caddy/Caddyfile << CADDYEOF
${DOMAIN} {
  tls admin@${DOMAIN}
  reverse_proxy /api/* 127.0.0.1:18888 {
    header_up X-Real-IP {http.request.header.CF-Connecting-IP}
  }
  handle {
    root * /srv/public
    try_files {path} /index.html
    file_server
  }
}
CADDYEOF
fi

cat > docker-compose.yml << DOCKEREOF
services:
  backend:
    image: debian:bookworm-slim
    network_mode: host
    restart: always
    env_file:
      - ./panel.env
    volumes:
      - ${INSTALL_DIR}/backend:/app/backend:ro
      - ${INSTALL_DIR}/data:/data
    command: /app/backend
    environment:
      - LISTEN=127.0.0.1:18888
      - DATABASE=sqlite3:///data/data.db
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
      - ${INSTALL_DIR}/public:/srv/public:ro
      - ${INSTALL_DIR}/caddy/Caddyfile:/etc/caddy/Caddyfile:ro
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
"${COMPOSE[@]}" down 2>/dev/null || true
"${COMPOSE[@]}" up -d

PROTO="https://"
ADDR="${DOMAIN}"
if [ -z "$DOMAIN" ] || is_ip "$DOMAIN"; then
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
if [ -n "$ADMIN_PWD" ]; then
  echo -e "${GREEN}  管理员密码使用安装时输入的值（不会回显）${NC}"
else
  echo -e "${GREEN}  已保留原管理员密码${NC}"
fi
echo -e "${GREEN}========================================${NC}"
echo ""
NODE_PROTO="https://"
[ -z "$DOMAIN" ] || is_ip "$DOMAIN" && NODE_PROTO="http://"
echo "节点客户端安装命令 (在入口机运行):"
echo -e "  ${YELLOW}ARCH=\$(case \$(uname -m) in x86_64|amd64) echo amd64;; aarch64|arm64) echo arm64;; esac); curl -fL https://github.com/${REPO}/releases/latest/download/nodeclient-linux-\${ARCH}.tar.gz -o /tmp/nc.tar.gz && tar xzf /tmp/nc.tar.gz -C /usr/local/bin/ && chmod +x /usr/local/bin/nodeclient${NC}"
echo ""
echo "对接入口机:"
echo -e "  ${YELLOW}nodeclient --server ${NODE_PROTO}${ADDR} --token <节点令牌> --device <设备组ID>${NC}"
