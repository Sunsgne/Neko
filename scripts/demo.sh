#!/usr/bin/env bash
# Neko SD-WAN 本地一键 Demo
# 启动后端 API（内存仓储 + 演示数据）与前端控制台，无需任何外部依赖。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
API_PORT="${API_PORT:-8080}"
WEB_PORT="${WEB_PORT:-3000}"

cleanup() {
  echo ""
  echo "==> 停止 Demo..."
  [[ -n "${API_PID:-}" ]] && kill "$API_PID" 2>/dev/null || true
  [[ -n "${WEB_PID:-}" ]] && kill "$WEB_PID" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

echo "==> 编译并启动后端 API (端口 ${API_PORT}, 演示数据已开启)..."
cd "$ROOT/backend"
go build -o bin/api ./cmd/api
NEKO_HTTP_ADDR=":${API_PORT}" NEKO_SEED=true NEKO_STORE=memory NEKO_LOG_LEVEL=info \
  ./bin/api &
API_PID=$!

echo "==> 等待后端就绪..."
for i in $(seq 1 30); do
  if curl -sf "http://localhost:${API_PORT}/healthz" >/dev/null 2>&1; then
    echo "    后端就绪: http://localhost:${API_PORT}"
    break
  fi
  sleep 0.5
done

echo "==> 安装前端依赖（首次较慢）并启动控制台 (端口 ${WEB_PORT})..."
cd "$ROOT/web"
if [[ ! -d node_modules ]]; then
  npm install
fi
NEXT_PUBLIC_API_BASE_URL="http://localhost:${API_PORT}" \
  npm run dev -- --port "${WEB_PORT}" &
WEB_PID=$!

cat <<EOF

────────────────────────────────────────────────────────
  Neko SD-WAN Demo 已启动
    控制台:  http://localhost:${WEB_PORT}
    API:     http://localhost:${API_PORT}/api/v1
    健康:    http://localhost:${API_PORT}/healthz

  演示数据已加载：3 租户 / 5 设备 / 5 链路 / 3 告警 / 6 DNS
  按 Ctrl+C 停止。
────────────────────────────────────────────────────────
EOF

wait
