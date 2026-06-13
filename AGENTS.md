# AGENTS.md — Neko SD-WAN 平台 · AI 自动开发说明书

> 本文件是**唯一权威开发说明书**。任何编码 AI（Cursor / Trae / Claude Code / Factory / Codex 等）在本仓库工作时，**必须先完整阅读本文件**，然后从 `docs/TASKS.md` 任务队列中领取任务并持续开发。
>
> 本文件是自包含的：不依赖任何此前的零散对话或补充说明。

---

## 0. 给 AI 的最高指令（务必遵守）

1. **持续开发，不要停。** 完成一个任务后，**立即**从 `docs/TASKS.md` 顶部领取下一个未完成任务继续开发，直到队列清空。
2. **不要每完成一步就询问"是否继续"。** 只在遇到**真正阻塞**（缺少凭据、需求自相矛盾、会造成数据破坏）时才提问。其余情况一律自行做出合理决策并记录到 `docs/DECISIONS.md`。
3. **不允许只搭骨架就停止。** 每个任务都要交付**可编译、可运行、有测试**的纵向切片（vertical slice），而非空壳。
4. **小步提交。** 每个逻辑变更一个 commit，提交信息使用 Conventional Commits（`feat:` `fix:` `chore:` `docs:` `test:` `refactor:`）。
5. **节省 Token。** 复用本说明书与现有代码约定，不要重复粘贴大段上下文；优先编辑现有文件而非新建；只读取必要文件。
6. **保持主线绿色。** 合并进主线前必须 `make check`（lint + vet + 单元测试 + 前端构建）全部通过。
7. **每个任务结束更新三件事：** `docs/TASKS.md`（勾选完成 / 追加新任务）、`docs/DECISIONS.md`（关键决策）、`CHANGELOG.md`（变更摘要）。

---

## 1. 产品愿景

**Neko** 是一个基于 **RouterOS（MikroTik）** 的 **SD-WAN 平台**，核心业务是：

- **网络加速**：智能选路、DNS 调度、链路质量优化（面向中国复杂网络环境，多运营商、多 POP）。
- **企业组网**：跨地域站点互联（Site-to-Site）、Overlay 隧道、动态路由（OSPF / BGP）。

平台提供两类使用者：

| 角色 | 端 | 能力 |
| --- | --- | --- |
| 平台运营方（我们） | **运营管理端 (Admin / Operator)** | 全局设备、租户、POP、镜像、模板、计费、审计、监控大盘 |
| 企业客户（租户） | **租户端 (Tenant)** | 自有站点 / 设备 / 链路 / 路由策略 / DNS / 监控（严格多租户隔离） |

---

## 2. 技术栈（已最终确定，不得擅自更换）

| 层 | 技术 | 用途 |
| --- | --- | --- |
| 后端控制平面 / Worker | **Go 1.22+** | API、设备编排、SNMP、探测、DNS、升级、路由计算 |
| Web 控制台 | **TypeScript + Next.js 14 (App Router)** | 运营端 + 租户端统一前端 |
| 核心数据库 | **PostgreSQL 16** | 业务数据、多租户隔离（行级 + schema 约定） |
| 缓存 / 锁 | **Redis 7** | 缓存、分布式锁、限流、会话 |
| 消息 / 任务 | **NATS JetStream** | 异步事件、任务队列、设备上报通道 |
| 时序指标 | **VictoriaMetrics** | 接口流量、链路延迟/丢包/抖动、资源监控 |
| 可观测性 | **OpenTelemetry** | 日志、指标、链路追踪（OTLP 导出） |

**语言选型理由（已决策，见 `docs/DECISIONS.md`）：** Go 拥有强并发模型、原生 goroutine 适合大规模并发 SNMP/探测/SSH 编排、静态编译易部署、丰富的网络库（gosnmp、crypto/ssh）；Next.js 提供现代化 SSR/RSC、优秀 DX 与一流 UI 生态。

---

## 3. 系统架构

```
                        ┌─────────────────────────────────────────┐
                        │            Web Console (Next.js)           │
                        │   Admin Portal   |   Tenant Portal         │
                        └───────────────┬───────────────────────────┘
                                        │ HTTPS / REST + SSE
                        ┌───────────────▼───────────────────────────┐
                        │        API Gateway (Go, chi/gin)            │
                        │  authn/authz · RBAC · multi-tenant scoping  │
                        └───┬───────────┬───────────┬───────────┬─────┘
                            │           │           │           │
              ┌─────────────▼┐ ┌────────▼─────┐ ┌───▼──────┐ ┌──▼───────────┐
              │ Inventory &  │ │ Config Engine│ │ Routing  │ │ Monitoring   │
              │ Onboarding   │ │ DesiredState │ │ OSPF/BGP │ │ SNMP/Probe   │
              └──────┬───────┘ └──────┬───────┘ └────┬─────┘ └──────┬───────┘
                     │                │              │              │
                 ┌───▼────────────────▼──────────────▼──────────────▼───┐
                 │                  NATS JetStream (bus)                  │
                 └───┬────────────────┬──────────────┬──────────────┬───┘
                     │                │              │              │
              ┌──────▼──────┐  ┌──────▼──────┐ ┌─────▼──────┐ ┌────▼────────┐
              │ Device Worker│  │ SNMP Poller │ │ Probe      │ │ Upgrade /   │
              │ (SSH/API)    │  │             │ │ Worker     │ │ Init Worker │
              └──────┬───────┘  └──────┬──────┘ └─────┬──────┘ └────┬────────┘
                     │                 │              │             │
                     ▼                 ▼              ▼             ▼
                 RouterOS 设备 (RB / CHR / x86)  ·  Agent on POP / CPE
                     ▲
          datastores: PostgreSQL · Redis · VictoriaMetrics · OTel Collector
```

**控制 / 数据分离原则**：控制平面下发"期望状态"；数据平面（RouterOS 设备 + POP）执行转发与本地快速切换；本地探测在毫秒级故障时**先本地切换**，再上报中心做全局决策。

---

## 4. 功能域（Feature Domains）

下列每个域都必须实现，对应 `backend/internal/<domain>` 与 `docs/TASKS.md` 中的史诗（Epic）。

### 4.1 多租户与 RBAC
- 租户（Tenant）、用户（User）、角色（Role）、API Token。
- 运营端可跨租户；租户端严格隔离（所有查询强制带 `tenant_id`，DB 层启用 RLS）。
- 审计日志（谁、何时、对哪个对象、做了什么、前后值）。

### 4.2 设备纳管与型号识别（Onboarding & Capability）
> 对应原始需求 #5。**核心：绝不只按型号下发固定脚本。**

- **自动识别**：通过 RouterOS API / SSH 读取：
  - 平台类型：**RouterBOARD / CHR / x86**。
  - `routeros-version`、架构（arm/arm64/mipsbe/tile/x86_64）、已装软件包列表、**License Level**、**Device Mode**（`/system/device-mode`）。
  - 接口清单及**接口能力**（是否支持 SFP/SFP+、PoE、wireless、L2/L3、速率）。
- **设备能力矩阵（Capability Matrix）**：将上述信息归一化为结构化能力集合，存库；下发决策基于**能力**而非型号字符串。
- **设备认证状态（Trust State）**：`untrusted → discovered → authenticated → enrolled → managed`；记录指纹（identity、serial、公钥）。
- 支持**纳管已有设备**（接管现网设备，先只读审计、再渐进式接管）。

### 4.3 配置引擎（Desired State Config Engine）
> 对应原始需求 #5。

- **Desired State**：租户/运营定义"期望配置"（声明式），引擎负责把设备收敛到该状态。
- **Diff 计算**：下发前对比 `running` 与 `desired`，生成最小变更集。
- **风险分级**：每个变更标注风险（low/medium/high/critical），如改动管理接口/路由/防火墙为高危。
- **管理通道保护**：任何下发不得切断管理通道；高危变更使用 **commit-confirm / safe-mode / rollback-timer**（下发后 N 秒内未确认自动回滚）。
- **自动验证**：下发后跑校验探针（连通性、路由、接口状态）确认成功。
- **配置回滚**：保留配置快照，失败自动回滚到上一个已知良好版本。
- **批量 Canary 灰度**：分批（1 台 → 5% → 25% → 100%），每批通过验证才进入下一批。

### 4.4 SD-WAN 组网与动态路由
> 对应原始需求 #4。**平台原生实现路由编排。**

- **Overlay**：站点间隧道（WireGuard / IPIP / EoIP / GRE，依设备能力选择）。
- **静态路由**：明细 + 默认路由 + 带距离/标记。
- **OSPF**：area 规划、接口 cost、被动接口、认证。
- **BGP**：**eBGP + iBGP**，**双 POP**，**Route Reflector**，**BFD** 快速收敛。
- **路由策略**：路由过滤（filter/prefix-list）、路由发布（advertise）、**路由汇总（aggregation）**、**重分发（redistribute）**。
- **防路由泄漏**：community 标记、ingress/egress 过滤、per-tenant VRF/路由表隔离，禁止租户路由互相泄漏。

### 4.5 SNMP 原生集成
> 对应原始需求 #3。**平台自身实现 SNMP，不仅是外部 LibreNMS 集成。**

- **SNMP 引擎**：v2c + **v3**（authPriv），基于 `gosnmp`。
- **设备发现**：网段扫描 + sysObjectID/sysDescr 识别。
- **轮询**：接口流量（ifHCInOctets/ifHCOutOctets）、接口状态、CPU、内存、温度、磁盘。
- **接口流量**：计算 bps、利用率、错误/丢弃率，写入 VictoriaMetrics。
- **资源监控**：CPU/内存/温度阈值告警。
- **Trap 接收**：监听 162，解析 trap，触发告警/事件。
- **告警**：规则引擎（阈值、持续时间、抑制、去重、升级），通知渠道（Webhook/邮件/钉钉/企微）。

### 4.6 版本与初始化（Lifecycle）
> 对应原始需求 #6、#9。

- **版本验证**：读取并校验当前 RouterOS 版本、channel、是否有可用更新。
- **RouterOS 升级**：受控升级（下载校验 → 灰度 → 升级 → 重启 → 验证 → 回滚预案）。
- **RouterBOOT 升级**：`/system/routerboard upgrade` 编排，注意需二次重启。
- **设备初始化**：开局模板（身份、管理用户/密钥、SNMP、NTP、时区、基础防火墙、管理通道、上联）。
- **接管已有设备**：从只读到托管的渐进流程。

### 4.7 DNS 管理与中国区调度
> 对应原始需求 #6、#9。**面向中国加速。**

- **DNS 服务器池**：管理大量上游 DNS（运营商/公共/自建），健康检查（解析正确性 + 延迟）。
- **调度策略**：按**地域 + 运营商（电信/联通/移动/教育网）** 选择最优 DNS；支持 ECS（EDNS Client Subnet）。
- **下发**：将选定 DNS 配置下发到 RouterOS（`/ip/dns`）或租户站点；支持分流（domain-list → 不同上游）。
- **可观测**：解析成功率、延迟、命中分布。

### 4.8 链路质量监控、上报与切换
> 对应原始需求 #7、#9。

- **指标**：**延迟、丢包、抖动**。
- **探测**：**ICMP、TCP、HTTP、HTTPS、DNS** 多类型主动探测。
- **链路评分（Link Score）**：综合延迟/丢包/抖动加权打分。
- **上报**：设备/POP 本地 Agent 周期上报到中心（NATS），中心入 VictoriaMetrics。
- **切换**：
  - **本地快速切换**：设备/Agent 在本地按评分阈值即时切换（主备/多线）。
  - **中心全局切换**：平台基于全局视图做跨 POP / 跨站点策略切换。
- **主备线路** + **自动恢复**（线路恢复后按策略回切）。
- **防震荡（flap damping）**：滞后阈值、最小驻留时间、抑制窗口，避免频繁抖动切换。

### 4.9 监控大盘与可观测性
- 运营大盘（全局健康、设备数、告警、流量 Top）。
- 租户大盘（自有站点/链路/DNS/告警）。
- 全链路 OTel：trace/metrics/logs 统一。

---

## 5. 现代化 Web UI 设计规范
> 对应原始需求 #10。体现高端、现代、专业的网络控制平台风格。

- **风格**：暗色优先（dark-first）的专业控制台风格，可切换浅色；强调数据密度与清晰层级。
- **设计语言**：玻璃拟态/微噪声背景克制使用；圆角 `rounded-xl`；柔和阴影；强调色用于状态（绿=健康、黄=告警、红=故障、蓝=信息）。
- **技术**：
  - **Tailwind CSS** + **shadcn/ui**（Radix 基础）组件体系。
  - **lucide-react** 图标，**Recharts / visx** 图表，**TanStack Query** 数据获取，**TanStack Table** 表格。
  - 拓扑可视化用 **React Flow**。
- **布局**：左侧可折叠导航 + 顶部命令面板（⌘K）+ 主内容区卡片网格。
- **可访问性**：键盘可达、对比度达标、语义化。
- **体验**：骨架屏、乐观更新、实时（SSE/WebSocket）状态、空态/错误态精心设计。
- **响应式**：桌面优先，平板可用。

详见 `docs/DESIGN.md`（UI 详规与 token）。

---

## 6. 仓库结构

```
.
├── AGENTS.md                 # 本说明书（权威）
├── README.md                 # 项目总览与快速开始
├── CHANGELOG.md              # 变更记录（每任务更新）
├── Makefile                  # 统一构建/检查入口
├── docker-compose.yml        # 本地全栈依赖
├── .env.example              # 环境变量样例
├── docs/
│   ├── TASKS.md              # 任务队列（AI 持续领取）
│   ├── DECISIONS.md          # 架构决策记录 (ADR)
│   ├── DESIGN.md             # UI/UX 详规
│   ├── API.md                # API 约定
│   └── ARCHITECTURE.md       # 架构细节
├── backend/                  # Go 控制平面 + workers
│   ├── go.mod
│   ├── cmd/
│   │   ├── api/              # API 服务入口
│   │   └── worker/           # 后台 worker 入口
│   ├── internal/
│   │   ├── config/           # 配置加载
│   │   ├── httpapi/          # HTTP 路由/handler/中间件
│   │   ├── observability/    # 日志/OTel
│   │   ├── tenant/           # 多租户与 RBAC
│   │   ├── inventory/        # 设备纳管与能力矩阵
│   │   ├── configengine/     # Desired State 引擎
│   │   ├── routing/          # OSPF/BGP/静态路由编排
│   │   ├── snmp/             # SNMP 引擎
│   │   ├── dns/              # DNS 调度
│   │   ├── linkqos/          # 链路质量与切换
│   │   ├── lifecycle/        # 版本/升级/初始化
│   │   └── store/            # 数据访问 (pgx) + migrations
│   └── migrations/           # SQL 迁移
└── web/                      # Next.js 控制台
    ├── package.json
    ├── app/                  # App Router 页面
    ├── components/
    └── lib/
```

---

## 7. 开发与运行约定

```bash
# 启动依赖（Postgres/Redis/NATS/VictoriaMetrics/OTel）
docker compose up -d

# 后端
cd backend && go run ./cmd/api          # API 服务
cd backend && go run ./cmd/worker       # worker

# 前端
cd web && npm install && npm run dev

# 统一检查（合并主线前必须通过）
make check                              # = backend lint/vet/test + web build
```

- **配置**：12-factor，全部走环境变量（见 `.env.example`），禁止硬编码密钥。
- **数据库迁移**：仅追加（forward-only），文件名 `NNNN_description.sql`。
- **API**：REST + JSON，资源路径 `/api/v1/...`；错误用统一 envelope；分页 `?page&page_size`。
- **安全**：所有写操作鉴权 + 租户作用域校验；SSH/SNMP/API 凭据加密存储（age/KMS）；最小权限。

---

## 8. 安全基线
- 传输全程 TLS；设备凭据静态加密；密钥不入库明文。
- RBAC + 多租户强隔离（PostgreSQL RLS）。
- 全量审计日志，不可篡改（追加写）。
- 高危配置下发的 commit-confirm 与自动回滚（保护管理通道）。
- 依赖与镜像安全扫描；最小基础镜像。

---

## 9. 测试与质量门禁
- Go：`go vet`、`golangci-lint`、`go test ./...`（单元 + 关键集成用 testcontainers 可选）。
- Web：`tsc --noEmit`、`eslint`、`next build`。
- 每个功能域至少覆盖：模型/服务单元测试 + 一条 happy-path 集成。
- CI（GitHub Actions）跑 `make check`。

---

## 10. 持续开发工作流（AI 循环）

```
loop:
  1. 读 docs/TASKS.md，取最上面一个未完成任务
  2. 实现该任务的纵向切片（代码 + 测试）
  3. 本地 make check 通过
  4. git commit（Conventional Commits）
  5. 更新 docs/TASKS.md（勾选/追加）、CHANGELOG.md、docs/DECISIONS.md（如有决策）
  6. git push
  7. 回到第 1 步 —— 不要停，不要问"是否继续"
直到队列清空；清空后，依据 §4 功能域补充新任务继续深化。
```

> **再次强调：不允许只完成骨架就停止；不允许每步都询问是否继续；自动读取任务队列并继续下一任务。**
