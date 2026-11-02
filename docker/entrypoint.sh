#!/bin/sh
# 容器入口：以 root 修正挂载目录属主后，降权到 UID 10001 运行宿主。
# 命名卷与 bind mount 默认由 root 创建，而宿主固定以 mica(10001) 运行，
# 不修正会出现 "mkdir /data/tmp: permission denied"。
set -eu

APP_UID=10001
APP_GID=10001

# 递归修正属主；失败不阻断启动（如只读挂载）。
fix_owner() {
	[ -d "$1" ] || return 0
	chown -R "$APP_UID:$APP_GID" "$1" 2>/dev/null || true
}

# 只修正顶层属主，避免对 NAS 上的大目录做递归 chown。
fix_owner_top() {
	[ -d "$1" ] || return 0
	chown "$APP_UID:$APP_GID" "$1" 2>/dev/null || true
}

if [ "$(id -u)" = "0" ] && [ "${MICA_NO_CHOWN:-0}" != "1" ]; then
	fix_owner /data
	fix_owner /app/modules
	fix_owner_top /media
	fix_owner_top /downloads
fi

if [ "$(id -u)" = "0" ]; then
	exec setpriv --reuid="$APP_UID" --regid="$APP_GID" --clear-groups /app/mica-home "$@"
fi

exec /app/mica-home "$@"
