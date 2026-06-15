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
- [x] T1.1 pgx 接入 + PostgresStore（tenants/devices 仓储）+ 嵌入式迁移执行器（NEKO_STORE=postgres）
- [x] T1.2 登录与鉴权全链路：users + session + /auth/login·me·logout + 前端登录页/路由保护/退出登录
- [x] T1.3 PostgreSQL 行级安全 (RLS) 多租户隔离（0002_rls.sql：tenant_isolation 策略 + current_tenant() GUC）
- [x] T1.4 审计日志：append-only 记录器 + 写操作埋点（create/trust_change）+ /api/v1/audit 查询 API

## Epic 2 — 设备纳管与能力矩阵
- [x] T2.1 RouterOS v7 REST 客户端（RestCollector，net/http + Basic Auth + 自签 TLS 容错，解析 resource/routerboard/package/license/device-mode/interface），已接入 inventory
- [x] T2.2 型号/版本/架构/包/License/Device Mode 识别（routeros.Detect + 测试）
- [x] T2.3 接口与接口能力发现，归一化能力矩阵（store.CapabilityMatrix）
- [x] T2.4 Trust State 状态机 + 设备纳管 Detect/SetTrustState + API 端点（含降级）

## Epic 3 — 配置引擎 (Desired State)
- [x] T3.1 Desired State 数据模型（Statement/State/Plan/Change）
- [x] T3.2 Diff 计算（属性级，确定性排序）+ 风险分级（路径基线 + 删除升级 + 管理通道保护=critical）
- [x] T3.3 下发执行器（configengine.Execute：snapshot→diff→风险闸门→commit-confirm 下发；Applier 接口含 device-side auto-revert 保护管理通道）
- [x] T3.4 自动验证（Verifier 接口）+ 验证失败自动 Restore 回滚
- [x] T3.5 批量 Canary 灰度编排（PlanCanaryBatches：1 台→5%→25%→100%，全覆盖）

## Epic 4 — SD-WAN 组网与动态路由
- [x] T4.1 Overlay 隧道编排（SelectTunnelType 按能力选 WireGuard/EoIP/GRE/IPIP；BuildTunnelState 生成 wireguard+peer / 隧道接口 + /ip/address 入 VRF）
- [x] T4.2 静态路由 + OSPF 编排 + 重分发（BuildState 生成 /ip/route、/routing/ospf、/routing/filter/rule 重分发规则）
- [x] T4.3 BGP（eBGP/iBGP 自动分类）+ Route Reflector client + BFD + 双 POP 邻居建模与语句生成
- [x] T4.4 路由策略：汇总（aggregate）+ 重分发建模 + 防泄漏校验（VRF/community 必填、eBGP 强制 import/export 过滤、重分发强制过滤、iBGP 全互联告警）

## Epic 5 — SNMP 原生引擎
- [x] T5.1 gosnmp 引擎（v2c + v3 authPriv，Get 标准 OID，凭据管理）
- [x] T5.2 设备发现（Discover：并发网段扫描 + sysDescr/sysObjectID/sysName 识别）
- [x] T5.3 轮询计算：接口计数器 → bps（32 位回绕/64 位重置处理）+ 利用率（VictoriaMetrics 写入见 T8.2 Reporter）
- [x] T5.4 Trap 接收器（TrapListener :162 + 解析为归一化 Trap）
- [x] T5.5 告警引擎 + Manager：阈值/for-duration/去重/firing↔resolved + 抑制（同设备高severity 抑制低severity）+ 升级（持续未恢复重通知）+ Notifier 通知渠道接口

## Epic 6 — 版本与初始化
- [x] T6.1 RouterOS 版本解析与比较（数值非字典序）+ NeedsUpgrade（lifecycle.CompareVersions）
- [x] T6.2 RouterOS 受控升级步骤编排（下载→校验→升级→重启→验证→健康检查）
- [x] T6.3 RouterBOOT 升级编排（仅 RouterBOARD，排在 RouterOS 之后 + 二次重启）
- [x] T6.4 设备初始化开局模板（identity/时区/NTP/SNMP/管理防火墙，生成 Desired State）

## Epic 7 — DNS 管理与中国区调度
- [x] T7.1 DNS 服务器池模型 + 健康检查器（Checker：直连各 DNS 解析探针域名，测延迟/正确性，并发 CheckAll）
- [x] T7.2 地域+运营商调度策略 + ECS（dns.Select 加权评分 + 公共兜底 + 确定性排序）
- [x] T7.3 下发到 RouterOS（BuildConfig 生成 /ip/dns + 分流 forwarders/static FWD）+ 解析质量可观测（HealthResult 延迟/成功率）
- [x] T7.4 DoH（DNS over HTTPS）支持：服务器池 kind=udp/doh，下发 use-doh-server+verify-doh-cert

## Epic 11 — 国内外加速（chnroutes 路由表分流）
- [x] T11.1 chnroutes 包：拉取/解析/缓存 chnroutes2 国内网段表（自定义源 URL）
- [x] T11.2 accel.BuildChinaSplitScript：国内网段→本地 WAN，海外 0.0.0.0/1+128.0.0.0/1→隧道（幂等 .rsc）
- [x] T11.3 routeros.Client.RunScript：上千条路由打包为单脚本一次性安装+执行（免登录、免逐条 REST）
- [x] T11.4 API + 前端「站点编排」国内外分流模式：状态/刷新/预览脚本/一键下发

## Epic 13 — 真实链路质量监控（设备实测）
- [x] T13.1 链路模型 + 仓储（memory+pg，迁移 0010）：绑定设备 + 探测目标
- [x] T13.2 routeros.Client.Ping（设备 /ping）+ linkqos.Aggregate/Status
- [x] T13.3 inventory.MeasureLink + Worker 60s 探测循环 + 即时探测 API
- [x] T13.4 API GET/POST/DELETE /links + /probe；种子改持久化
- [x] T13.5 前端「链路质量」可管理（添加/删除/即时探测）+ 实测展示；rosim /ping

## Epic 12 — 全功能远程配置（所有 RouterOS 菜单）
- [x] T12.1 routeros.Catalog：WebFig 全菜单配置段目录 + ValidPath 校验
- [x] T12.2 inventory.REST{List,Create,Update,Delete,Set} + Client.Set：用托管凭据免登录读写任意段
- [x] T12.3 API：GET /config/catalog + GET/POST/PATCH/DELETE /devices/{id}/rest（含 singleton set，审计）
- [x] T12.4 前端「远程配置」页：菜单树 + 配置项表格 + 通用键值表单（新增/编辑/删除/单例设置）
- [x] T12.5 rosim 支持 /set；端到端验证多菜单 CRUD + 单例 set

## Epic 8 — 链路质量监控、上报与切换
- [x] T8.1 探测引擎（probe 包：ICMP/TCP/HTTP/HTTPS/DNS，Run 聚合延迟/丢包/抖动）
- [x] T8.2 链路评分（ScoreResult）+ 上报通道（VictoriaMetricsReporter，Influx line-protocol → /write）
- [x] T8.3 切换决策（linkqos.Controller）：主备 + 自动失败回切 + 防震荡（滞后阈值 + MinDown/MinUp + 最小驻留 MinDwell）

## Epic 9 — 可观测性与大盘
- [x] T9.1 可观测性：metrics 注册表 + Prometheus /metrics 端点（VictoriaMetrics 可抓取）+ 请求埋点中间件 + 结构化日志（request_id）；OTLP trace 导出为后续增强
- [x] T9.2 运营/租户大盘前端（按角色标题，KPI 取自真实 API：设备/在线率/告警/链路均分 + 质量分布）
- [x] T9.3 拓扑可视化（自绘 SVG 中心辐射图，按链路状态着色/评分调线宽/Overlay 虚线，无额外依赖）

## Epic 10 — 交付与运维
- [x] T10.1 GitHub Actions CI（make check）
- [x] T10.2 容器镜像与部署清单（backend distroless 多阶段 Dockerfile + web Next standalone Dockerfile + docker-compose.deploy.yml 全栈编排）
- [x] T10.3 端到端演示数据与种子脚本（catalog + seed + make demo / scripts/demo.sh）
