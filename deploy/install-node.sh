#!/bin/bash
set -euo pipefail

REPO="CaydenTrudell7194/anotherpass"
VERSION="latest"
SERVER=""
ENROLL=""

while [ $# -gt 0 ]; do
  case "$1" in
    --server) SERVER="${2:-}"; shift 2 ;;
    --enroll) ENROLL="${2:-}"; shift 2 ;;
    *) echo "未知参数: $1"; exit 1 ;;
  esac
done

if [ "${EUID}" -ne 0 ] || [ -z "$SERVER" ] || [ -z "$ENROLL" ]; then
  echo "用法: bash install-node.sh --server https://panel.example.com --enroll <一次性注册码>"
  exit 1
fi
if ! [[ "$SERVER" =~ ^https?://[A-Za-z0-9._:\[\]/-]+$ ]] || ! [[ "$ENROLL" =~ ^[a-f0-9]{48}$ ]]; then
  echo "面板地址或一次性注册码格式无效"
  exit 1
fi

case "$(uname -m)" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "不支持的架构: $(uname -m)"; exit 1 ;;
esac

mkdir -p /etc/forward-node
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT
if [ "$VERSION" = "latest" ]; then
  ASSET_URL="https://github.com/${REPO}/releases/latest/download/nodeclient-linux-${ARCH}.tar.gz"
else
  ASSET_URL="https://github.com/${REPO}/releases/download/${VERSION}/nodeclient-linux-${ARCH}.tar.gz"
fi
curl --proto '=https' --tlsv1.2 -fL "$ASSET_URL" -o "$TMP_DIR/nodeclient.tar.gz"
tar xzf "$TMP_DIR/nodeclient.tar.gz" -C /usr/local/bin/
chmod 0755 /usr/local/bin/nodeclient

umask 077
/usr/local/bin/nodeclient --server "$SERVER" --enroll "$ENROLL" --output-config /etc/forward-node/config.json
chmod 0600 /etc/forward-node/config.json

cat > /etc/systemd/system/forward-node.service <<'EOF'
[Unit]
Description=Forward Panel Entry Node
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/nodeclient --config /etc/forward-node/config.json
Restart=always
RestartSec=5
LimitNOFILE=1048576
NoNewPrivileges=true
ProtectHome=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable forward-node
systemctl restart forward-node
echo "入口节点安装完成，查看日志: journalctl -u forward-node -f"
