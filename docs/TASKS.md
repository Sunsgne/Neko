# TASKS.md — 任务队列（AI 持续领取）

> 规则：从上往下领取**第一个未完成**任务；完成后勾选并提交；队列见底后按 `AGENTS.md` §4 追加深化任务。**不要停。**
>
> 状态图例：`[ ]` 待办 · `[~]` 进行中 · `[x]` 完成

## Epic 0 — 项目基线 (Bootstrap)
- [x] T0.1 编写 `AGENTS.md` 权威说明书
- [x] T0.2 仓库结构、根 README、.gitignore、Makefile、docker-compose、.env.example
- [x] T0.3 Go 后端骨架：config、结构化日志、HTTP server、健康检查、优雅退出
- [x] T0.4 PostgreSQL schema 与迁移（tenants/users/sites/devices/links/...）
- [x] T0.5 第一个纵向切片：Tenant + Device REST API（内存仓储，便于无 DB 运行）
- [x] T0.6 Next.js 控制台骨架 + 现代化 UI shell（仪表盘 / 设备 / 租户）

## Epic 1 — 多租户与 RBAC
- [ ] T1.1 pgx 接入 + 仓储实现（替换内存仓储，保留接口）
- [~] T1.2 Token 鉴权中间件（auth 包：Principal/operator-tenant 作用域、SHA-256 哈希存储、可选开关 NEKO_AUTH）；用户/角色 RBAC 细化待补
- [ ] T1.3 PostgreSQL 行级安全 (RLS) 多租户隔离
- [~] T1.4 审计日志模型与追加写记录器（audit 包，append-only）；写操作埋点与查询 API 待补

## Epic 2 — 设备纳管与能力矩阵
- [~] T2.1 RouterOS API/SSH 客户端封装（已定义 Collector 接口 + StaticCollector；REST 实现待补）
- [x] T2.2 型号/版本/架构/包/License/Device Mode 识别（routeros.Detect + 测试）
- [x] T2.3 接口与接口能力发现，归一化能力矩阵（store.CapabilityMatrix）
- [x] T2.4 Trust State 状态机 + 设备纳管 Detect/SetTrustState + API 端点（含降级）

## Epic 3 — 配置引擎 (Desired State)
- [x] T3.1 Desired State 数据模型（Statement/State/Plan/Change）
- [x] T3.2 Diff 计算（属性级，确定性排序）+ 风险分级（路径基线 + 删除升级 + 管理通道保护=critical）
- [ ] T3.3 下发执行器（commit-confirm / safe-mode / rollback-timer / 管理通道保护）
- [ ] T3.4 自动验证探针 + 自动回滚
- [ ] T3.5 批量 Canary 灰度编排

## Epic 4 — SD-WAN 组网与动态路由
- [ ] T4.1 Overlay 隧道编排（WireGuard/IPIP/EoIP/GRE 按能力）
- [~] T4.2 静态路由 + OSPF 编排（routing.Intent 模型 + BuildState 生成 /ip/route、/routing/ospf 语句）
- [x] T4.3 BGP（eBGP/iBGP 自动分类）+ Route Reflector client + BFD + 双 POP 邻居建模与语句生成
- [x] T4.4 路由策略：汇总（aggregate）+ 重分发建模 + 防泄漏校验（VRF/community 必填、eBGP 强制 import/export 过滤、重分发强制过滤、iBGP 全互联告警）

## Epic 5 — SNMP 原生引擎
- [ ] T5.1 gosnmp 引擎（v2c/v3）+ 凭据管理
- [ ] T5.2 设备发现（网段扫描 + sysObjectID 识别）
- [~] T5.3 轮询计算：接口计数器 → bps（含 32 位回绕/64 位重置处理）+ 利用率（snmp 包）；VictoriaMetrics 写入待补
- [ ] T5.4 Trap 接收器（:162）+ 解析
- [~] T5.5 告警规则引擎（alerting 包：阈值 + for-duration + 去重 + firing/resolved 转换 + 多 series 隔离）；抑制/升级/通知渠道待补

## Epic 6 — 版本与初始化
- [x] T6.1 RouterOS 版本解析与比较（数值非字典序）+ NeedsUpgrade（lifecycle.CompareVersions）
- [x] T6.2 RouterOS 受控升级步骤编排（下载→校验→升级→重启→验证→健康检查）
- [x] T6.3 RouterBOOT 升级编排（仅 RouterBOARD，排在 RouterOS 之后 + 二次重启）
- [x] T6.4 设备初始化开局模板（identity/时区/NTP/SNMP/管理防火墙，生成 Desired State）

## Epic 7 — DNS 管理与中国区调度
- [~] T7.1 DNS 服务器池模型（dns.Server）；健康检查器待补
- [x] T7.2 地域+运营商调度策略 + ECS（dns.Select 加权评分 + 公共兜底 + 确定性排序）
- [ ] T7.3 下发到 RouterOS（/ip/dns + 分流）+ 解析质量可观测

## Epic 8 — 链路质量监控、上报与切换
- [ ] T8.1 探测引擎（ICMP/TCP/HTTP/HTTPS/DNS）+ 延迟/丢包/抖动
- [~] T8.2 链路评分（linkqos.Score 加权延迟/丢包/抖动）；上报通道（NATS → VictoriaMetrics）待补
- [x] T8.3 切换决策（linkqos.Controller）：主备 + 自动失败回切 + 防震荡（滞后阈值 + MinDown/MinUp + 最小驻留 MinDwell）

## Epic 9 — 可观测性与大盘
- [ ] T9.1 OpenTelemetry（trace/metrics/logs）贯通
- [ ] T9.2 运营大盘 + 租户大盘前端
- [ ] T9.3 拓扑可视化（React Flow）

## Epic 10 — 交付与运维
- [x] T10.1 GitHub Actions CI（make check）
- [ ] T10.2 容器镜像与部署清单
- [x] T10.3 端到端演示数据与种子脚本（catalog + seed + make demo / scripts/demo.sh）
