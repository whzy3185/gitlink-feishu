#!/bin/sh
set -eu

SERVICE_USER="${SERVICE_USER:-gitlink-feishu-review}"
UNIT_SOURCE="${UNIT_SOURCE:-./gitlink-feishu-review.service}"
ENV_SOURCE="${ENV_SOURCE:-./gitlink-feishu-review.env.example}"

getent group "$SERVICE_USER" >/dev/null 2>&1 || groupadd --system "$SERVICE_USER"
id "$SERVICE_USER" >/dev/null 2>&1 || useradd --system --gid "$SERVICE_USER" --home /var/lib/gitlink-feishu-review --shell /usr/sbin/nologin "$SERVICE_USER"
install -d -m 0750 -o "$SERVICE_USER" -g "$SERVICE_USER" /var/lib/gitlink-feishu-review /var/backups/gitlink-feishu-review /var/log/gitlink-feishu-review
install -d -m 0750 -o root -g "$SERVICE_USER" /etc/gitlink-feishu-review
install -m 0644 "$UNIT_SOURCE" /etc/systemd/system/gitlink-feishu-review.service
if [ ! -f /etc/gitlink-feishu-review/review-service.env ]; then
  install -m 0600 -o root -g "$SERVICE_USER" "$ENV_SOURCE" /etc/gitlink-feishu-review/review-service.env
fi
systemctl daemon-reload
systemctl enable gitlink-feishu-review.service
printf '%s\n' 'Installed. Configure the protected environment and bindings files before starting.'
