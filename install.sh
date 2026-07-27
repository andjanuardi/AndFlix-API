#!/bin/bash
set -e

BINARY="./andflix-api"
INSTALL_DIR="/usr/local/bin"
SERVICE_NAME="andflix-api"
SERVICE_USER="root"
PORT="9996"

if [ ! -f "$BINARY" ]; then
  echo "Error: $BINARY not found. Run build script first."
  exit 1
fi

cp "$BINARY" "$INSTALL_DIR/$SERVICE_NAME"
chmod +x "$INSTALL_DIR/$SERVICE_NAME"

cat > /etc/systemd/system/$SERVICE_NAME.service <<EOF
[Unit]
Description=ANDFLIX API Service
After=network.target

[Service]
Type=simple
User=$SERVICE_USER
ExecStart=$INSTALL_DIR/$SERVICE_NAME $PORT
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable $SERVICE_NAME
systemctl restart $SERVICE_NAME

echo "ANDFLIX API Service installed and running on port $PORT"
