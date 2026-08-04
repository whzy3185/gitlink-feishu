#!/bin/sh
set -eu

systemctl stop gitlink-feishu-review.service 2>/dev/null || true
systemctl disable gitlink-feishu-review.service 2>/dev/null || true
rm -f /etc/systemd/system/gitlink-feishu-review.service
systemctl daemon-reload
printf '%s\n' 'Service unit removed. State, backups, bindings, and protected environment files were preserved.'
