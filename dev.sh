#!/usr/bin/env bash

# 这个脚本用于本地联调：
# 1. 先停止占用后端 / admin 端口的旧进程，避免重复启动时报 address already in use。
# 2. 支持 start / stop 两种参数，统一管理本地后端服务和 admin 前端。
# 3. 所有日志都会落到 .tmp/dev-logs，方便在 IDE 外单独查看排障信息。

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
BACKEND_DIR="$ROOT_DIR/backend"
ADMIN_DIR="$ROOT_DIR/admin"
LOG_DIR="$ROOT_DIR/.tmp/dev-logs"
BACKEND_PORT="8080"
ADMIN_PORT="5173"
ADMIN_PORT_RANGE_START="5173"
ADMIN_PORT_RANGE_END="5180"
BACKEND_LOG="$LOG_DIR/backend.log"
ADMIN_LOG="$LOG_DIR/admin.log"
COMMAND="${1:-start}"

mkdir -p "$LOG_DIR"

# stop_port_processes 会强制结束占用指定端口的旧进程。
# 这里按端口清理，而不是按进程名清理，避免误伤你机器上其他 Go / Node 项目。
stop_port_processes() {
  local port="$1"
  local pids

  pids="$(lsof -ti tcp:"$port" || true)"
  if [[ -z "$pids" ]]; then
    echo "[dev.sh] port $port is already free"
    return 0
  fi

  echo "[dev.sh] stopping processes on port $port: $pids"
  # 用户已经明确要求“先 stop 再启动”，这里直接结束旧进程，确保新进程可绑定端口。
  kill -9 $pids
}

stop_all() {
  stop_port_processes "$BACKEND_PORT"
  local port
  for ((port=ADMIN_PORT_RANGE_START; port<=ADMIN_PORT_RANGE_END; port++)); do
    stop_port_processes "$port"
  done
}

# ensure_admin_dependencies 只在 node_modules 不存在时执行 npm install，避免每次启动都重复安装。
ensure_admin_dependencies() {
  if [[ -d "$ADMIN_DIR/node_modules" ]]; then
    echo "[dev.sh] admin dependencies already installed"
    return 0
  fi

  echo "[dev.sh] installing admin dependencies"
  (cd "$ADMIN_DIR" && npm install)
}

# start_backend 使用 nohup 把后端放到后台，日志写入独立文件，便于持续查看服务端输出。
start_backend() {
  echo "[dev.sh] starting backend server"
  nohup bash -lc "cd '$BACKEND_DIR' && go run ./server/cmd/game-server" >"$BACKEND_LOG" 2>&1 &
  echo "[dev.sh] backend log: $BACKEND_LOG"
}

# start_admin 使用 nohup 启动 Vite 开发服务器，保持脚本退出后前端仍然运行。
start_admin() {
  echo "[dev.sh] starting admin frontend"
  nohup bash -lc "cd '$ADMIN_DIR' && npm run dev -- --host 0.0.0.0 --port $ADMIN_PORT --strictPort" >"$ADMIN_LOG" 2>&1 &
  echo "[dev.sh] admin log: $ADMIN_LOG"
}

print_next_steps() {
  cat <<MSG
[dev.sh] done
- backend: http://localhost:$BACKEND_PORT
- admin:   http://localhost:$ADMIN_PORT
- tail backend log: tail -f "$BACKEND_LOG"
- tail admin log:   tail -f "$ADMIN_LOG"
MSG
}

print_usage() {
  cat <<MSG
usage:
  ./dev.sh start   # 停掉旧进程后重新启动后端和 admin
  ./dev.sh stop    # 停掉后端和 admin
MSG
}

case "$COMMAND" in
  start)
    stop_all
    ensure_admin_dependencies
    start_backend
    start_admin
    print_next_steps
    ;;
  stop)
    stop_all
    echo "[dev.sh] stopped backend and admin processes"
    ;;
  *)
    print_usage
    exit 1
    ;;
esac
