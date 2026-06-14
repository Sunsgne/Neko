# Neko — RouterOS SD-WAN 平台 🐱

基于 **RouterOS (MikroTik)** 的 SD-WAN 平台，核心业务是**网络加速**与**企业组网**，面向中国复杂网络环境。提供**运营管理端**与**租户端**双门户。

> 🤖 **AI 开发者请先阅读 [`AGENTS.md`](./AGENTS.md)**，它是本仓库唯一权威开发说明书，并从 [`docs/TASKS.md`](./docs/TASKS.md) 持续领取任务。

## 能力概览
- 多租户 + RBAC，严格租户隔离
- 设备纳管：自动识别 RouterBOARD/CHR/x86、版本、架构、License、Device Mode、接口能力（能力矩阵）
- 配置引擎：Desired State + Diff + 风险分级 + commit-confirm 安全回滚 + Canary 灰度
- SD-WAN 组网：Overlay + 静态/OSPF/BGP（eBGP/iBGP/双 POP/RR/BFD）+ 路由策略与防泄漏
- 原生 SNMP：发现/轮询/接口流量/资源/Trap/告警
- 生命周期：RouterOS/RouterBOOT 升级、设备初始化、接管现网设备
- DNS 管理：大规模 DNS 池 + 中国地域/运营商调度
- 链路质量：延迟/丢包/抖动，多探测，本地+全局切换，主备/自动恢复/防震荡
- 可观测：OpenTelemetry + VictoriaMetrics 大盘

## 技术栈
Go 1.22+（后端/Worker） · Next.js 14 + TypeScript（Web） · PostgreSQL 16 · Redis 7 · NATS JetStream · VictoriaMetrics · OpenTelemetry

## 一键本地 Demo 🚀

无需数据库或任何外部依赖（后端使用内存仓储 + 演示数据）：

```bash
make demo
```

启动后访问：

- 控制台：http://localhost:3000
- API：http://localhost:8080/api/v1 （健康检查 `/healthz`）

演示数据包含 3 个租户、5 台设备（含能力矩阵）、5 条链路（含评分）、3 条告警、6 个 DNS 服务器。按 `Ctrl+C` 停止。

> 前置：本机已安装 Go 1.22+ 与 Node 20+。首次运行会自动 `npm install`（稍慢）。

## 快速开始（分步）

```bash
# 1) 启动依赖（可选，后端默认内存仓储可无依赖运行）
docker compose up -d

# 2) 后端 API（加演示数据用 make backend-run-seed）
cd backend && NEKO_SEED=true go run ./cmd/api
# → http://localhost:8080/healthz

# 3) 前端控制台
cd web && npm install && npm run dev
# → http://localhost:3000

# 统一检查
make check
```

## 目录结构
见 [`AGENTS.md` §6](./AGENTS.md) 与 [`docs/ARCHITECTURE.md`](./docs/ARCHITECTURE.md)。

## 文档
- [`AGENTS.md`](./AGENTS.md) — AI 自动开发说明书（权威）
- [`docs/TASKS.md`](./docs/TASKS.md) — 任务队列
- [`docs/ARCHITECTURE.md`](./docs/ARCHITECTURE.md) — 架构
- [`docs/DESIGN.md`](./docs/DESIGN.md) — UI/UX 规范
- [`docs/API.md`](./docs/API.md) — API 约定
- [`docs/DECISIONS.md`](./docs/DECISIONS.md) — 架构决策记录

## License
[MIT](./LICENSE)
