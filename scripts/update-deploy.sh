#!/usr/bin/env bash
# ────────────────────────────────────────────────────────────────────────────
# Neko SD-WAN 平台 · 远程 Docker 升级脚本
#
# 在已部署的服务器上拉取最新代码、更新基础镜像并重建应用容器。
#
# 用法（在仓库根目录，root 或 sudo）：
#   sudo bash scripts/update-deploy.sh
#
# 可选环境变量：
#   GIT_BRANCH=main           # 要拉取的分支（默认 main）
#   SKIP_PULL=1               # 跳过 git pull（仅重建镜像）
#   SKIP_BASE_PULL=1          # 跳过 docker compose pull（仅重建应用）
# ────────────────────────────────────────────────────────────────────────────
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

c_blue='\033[0;34m'; c_green='\033[0;32m'; c_yellow='\033[0;33m'; c_red='\033[0;31m'; c_reset='\033[0m'
log()  { echo -e "${c_blue}==>${c_reset} $*"; }
ok()   { echo -e "${c_green}✓${c_reset} $*"; }
warn() { echo -e "${c_yellow}!${c_reset} $*"; }
die()  { echo -e "${c_red}✗ $*${c_reset}" >&2; exit 1; }

SUDO=""
if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  command -v sudo >/dev/null 2>&1 || die "需要 root 权限或已安装 sudo"
  SUDO="sudo"
fi

command -v docker >/dev/null 2>&1 || die "未安装 Docker，请先运行 scripts/deploy-ubuntu.sh"
docker compose version >/dev/null 2>&1 || die "未安装 docker compose 插件"

COMPOSE=(docker compose -f docker-compose.yml -f docker-compose.deploy.yml)
GIT_BRANCH="${GIT_BRANCH:-main}"

# ── 1. 拉取最新代码 ──
if [[ -z "${SKIP_PULL:-}" ]]; then
  if [[ -d .git ]]; then
    log "拉取最新代码 (${GIT_BRANCH})..."
    git fetch origin "$GIT_BRANCH"
    git checkout "$GIT_BRANCH"
    git pull --ff-only origin "$GIT_BRANCH"
    ok "代码已更新：$(git log -1 --oneline)"
  else
    warn "非 git 仓库，跳过代码拉取"
  fi
else
  warn "SKIP_PULL=1，跳过 git pull"
fi

[[ -f .env ]] || die "未找到 .env，请先运行 scripts/deploy-ubuntu.sh 完成首次部署"

# ── 2. 更新基础镜像（Postgres/Redis/NATS/VM/OTel）──
if [[ -z "${SKIP_BASE_PULL:-}" ]]; then
  log "拉取最新基础镜像..."
  $SUDO "${COMPOSE[@]}" pull postgres redis nats victoriametrics otel-collector 2>/dev/null || \
    $SUDO "${COMPOSE[@]}" pull
  ok "基础镜像已更新"
else
  warn "SKIP_BASE_PULL=1，跳过基础镜像拉取"
fi

# ── 3. 重建并滚动升级应用 ──
log "重建并启动服务（api / worker / web / rosim）..."
$SUDO "${COMPOSE[@]}" up -d --build --remove-orphans

# ── 4. 等待 API 就绪 ──
set -a; . ./.env; set +a
API_PORT="${NEKO_API_PORT:-8080}"
log "等待 API 健康检查..."
ready=0
for i in $(seq 1 60); do
  if curl -fsS "http://localhost:${API_PORT}/healthz" >/dev/null 2>&1; then
    ready=1; break
  fi
  sleep 2
done
[[ "$ready" == "1" ]] && ok "升级完成，API 就绪" || warn "API 未在预期时间内就绪，请检查：docker compose logs api"

# ── 5. 输出当前版本 ──
VER="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
cat <<EOF

────────────────────────────────────────────────────────────
  🐱 Neko 远程 Docker 已升级
────────────────────────────────────────────────────────────
  版本:     ${VER}
  控制台:   http://localhost:${NEKO_WEB_PORT:-3000}
  API:      http://localhost:${API_PORT}/api/v1
  健康检查: http://localhost:${API_PORT}/healthz

  查看状态: docker compose -f docker-compose.yml -f docker-compose.deploy.yml ps
  查看日志: docker compose -f docker-compose.yml -f docker-compose.deploy.yml logs -f api
────────────────────────────────────────────────────────────
EOF
