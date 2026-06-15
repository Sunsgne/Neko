#!/usr/bin/env bash
# ────────────────────────────────────────────────────────────────────────────
# Neko SD-WAN 平台 · Ubuntu 24.04 一键部署脚本
#
# 用法（在服务器上，以 root 或具备 sudo 的用户运行）：
#   sudo bash scripts/deploy-ubuntu.sh
#
# 可选环境变量：
#   PUBLIC_HOST=1.2.3.4         # 对外访问地址（默认自动探测本机 IP）
#   ADMIN_EMAIL=you@corp.com    # 运营管理员账号（默认 admin@neko.io）
#   ADMIN_PASSWORD=...          # 管理员密码（默认随机生成）
#   NEKO_API_PORT=8080          # API 端口
#   NEKO_WEB_PORT=3000          # 控制台端口
#   WITH_DEMO=true|false        # 是否注入演示数据/账号（默认 true）
#   FORCE_ENV=1                 # 强制重新生成 .env（覆盖已有密钥）
#
# 脚本是幂等的：重复运行会复用已生成的 .env 并执行滚动升级。
# ────────────────────────────────────────────────────────────────────────────
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

# ── 日志助手 ──
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

# ── 1. 系统检查 ──
log "检查操作系统..."
if [[ -r /etc/os-release ]]; then
  . /etc/os-release
  if [[ "${ID:-}" == "ubuntu" && "${VERSION_ID:-}" == "24.04" ]]; then
    ok "Ubuntu 24.04 (Noble)"
  else
    warn "当前系统为 ${PRETTY_NAME:-未知}，脚本面向 Ubuntu 24.04，将尽力继续。"
  fi
else
  warn "无法识别系统版本，继续尝试。"
fi

# ── 2. 安装 Docker Engine + compose 插件 ──
install_docker() {
  log "安装 Docker Engine 与 compose 插件..."
  export DEBIAN_FRONTEND=noninteractive
  $SUDO apt-get update -y
  $SUDO apt-get install -y ca-certificates curl gnupg openssl
  $SUDO install -m 0755 -d /etc/apt/keyrings
  if [[ ! -f /etc/apt/keyrings/docker.asc ]]; then
    $SUDO curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
    $SUDO chmod a+r /etc/apt/keyrings/docker.asc
  fi
  local codename="${UBUNTU_CODENAME:-noble}"
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu ${codename} stable" \
    | $SUDO tee /etc/apt/sources.list.d/docker.list >/dev/null
  $SUDO apt-get update -y
  $SUDO apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
  $SUDO systemctl enable --now docker
}

if ! command -v docker >/dev/null 2>&1; then
  install_docker
else
  ok "Docker 已安装：$(docker --version)"
fi

if ! docker compose version >/dev/null 2>&1; then
  warn "未检测到 docker compose 插件，尝试安装..."
  install_docker
fi
ok "compose 可用：$(docker compose version | head -n1)"

# ── 3. 生成 .env（含随机密钥）──
rand() { openssl rand -hex "${1:-16}"; }
detect_ip() {
  local ip
  ip="$(ip route get 1.1.1.1 2>/dev/null | awk '{for(i=1;i<=NF;i++) if($i=="src"){print $(i+1); exit}}')"
  [[ -z "$ip" ]] && ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
  [[ -z "$ip" ]] && ip="localhost"
  echo "$ip"
}

API_PORT="${NEKO_API_PORT:-8080}"
WEB_PORT="${NEKO_WEB_PORT:-3000}"
PUBLIC_HOST="${PUBLIC_HOST:-$(detect_ip)}"
WITH_DEMO="${WITH_DEMO:-true}"

if [[ -f .env && -z "${FORCE_ENV:-}" ]]; then
  ok "复用已有 .env（如需重置请加 FORCE_ENV=1）"
else
  log "生成 .env（随机密钥）..."
  local_pg_pass="$(rand 24)"
  local_admin_pass="${ADMIN_PASSWORD:-$(rand 12)}"
  admin_email="${ADMIN_EMAIL:-admin@neko.io}"
  cat > .env <<EOF
# 由 scripts/deploy-ubuntu.sh 生成于 $(date -u +%FT%TZ)
NEKO_ENV=production
NEKO_STORE=postgres
NEKO_AUTH=on
NEKO_SEED=${WITH_DEMO}

NEKO_ADMIN_EMAIL=${admin_email}
NEKO_ADMIN_PASSWORD=${local_admin_pass}

POSTGRES_USER=neko
POSTGRES_PASSWORD=${local_pg_pass}
POSTGRES_DB=neko

NEKO_API_PORT=${API_PORT}
NEKO_WEB_PORT=${WEB_PORT}
NEKO_PUBLIC_API_URL=http://${PUBLIC_HOST}:${API_PORT}
EOF
  ok ".env 已生成（管理员密码已写入 .env，请妥善保存）"
fi

# 读取最终值用于展示
set -a; . ./.env; set +a
PUBLIC_HOST_SHOWN="${NEKO_PUBLIC_API_URL#http://}"; PUBLIC_HOST_SHOWN="${PUBLIC_HOST_SHOWN%%:*}"

# ── 4. 构建并启动全栈 ──
if [[ -d .git && -z "${SKIP_PULL:-}" ]]; then
  log "拉取最新代码..."
  git fetch origin main 2>/dev/null && git pull --ff-only origin main 2>/dev/null || warn "git pull 跳过（非 git 仓库或拉取失败）"
fi
log "构建并启动服务（首次构建较慢，请耐心等待）..."
$SUDO docker compose -f docker-compose.yml -f docker-compose.deploy.yml up -d --build

# ── 5. 等待 API 就绪 ──
log "等待 API 健康检查..."
ready=0
for i in $(seq 1 60); do
  if curl -fsS "http://localhost:${NEKO_API_PORT:-8080}/healthz" >/dev/null 2>&1; then
    ready=1; break
  fi
  sleep 2
done
[[ "$ready" == "1" ]] && ok "API 就绪" || warn "API 未在预期时间内就绪，请检查：docker compose logs api"

# ── 6. 输出访问信息 ──
cat <<EOF

────────────────────────────────────────────────────────────
  🐱 Neko SD-WAN 平台部署完成
────────────────────────────────────────────────────────────
  控制台:   http://${PUBLIC_HOST_SHOWN}:${NEKO_WEB_PORT:-3000}
  API:      http://${PUBLIC_HOST_SHOWN}:${NEKO_API_PORT:-8080}/api/v1
  健康检查: http://${PUBLIC_HOST_SHOWN}:${NEKO_API_PORT:-8080}/healthz
  指标:     http://${PUBLIC_HOST_SHOWN}:${NEKO_API_PORT:-8080}/metrics

  管理员账号: ${NEKO_ADMIN_EMAIL}
  管理员密码: ${NEKO_ADMIN_PASSWORD}
  （凭据保存在 ${REPO_ROOT}/.env）

  常用命令:
    docker compose -f docker-compose.yml -f docker-compose.deploy.yml ps
    docker compose -f docker-compose.yml -f docker-compose.deploy.yml logs -f api
    docker compose -f docker-compose.yml -f docker-compose.deploy.yml down
────────────────────────────────────────────────────────────
EOF
