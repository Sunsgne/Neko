# Changelog

本项目遵循 [Keep a Changelog](https://keepachangelog.com/) 与 [Conventional Commits](https://www.conventionalcommits.org/)。

## [Unreleased]

### Added
- **真实链路质量监控(real link-quality monitoring)**：替换原「链路质量」页的演示数据为**设备实测**。
  - 链路模型 + 仓储(memory+pg,迁移 0010 `links` 表)：每条链路绑定设备与探测目标(target)。
  - `routeros.Client.Ping`：调用设备自身 ping 工具(REST `POST /ping`)从**设备视角**测目标延迟；`linkqos.Aggregate`(延迟/抖动/丢包)+`Status`(up/degraded/down)+ 既有 `Score`。
  - `inventory.MeasureLink`：用托管凭据免登录从设备实测；Worker 每 60s 探测所有链路并落库;API 提供即时探测 `POST /links/{id}/probe`。
  - API：`GET/POST/DELETE /links` + `POST /links/{id}/probe`(全审计);种子链路改为持久化、绑定设备。
  - 前端「链路质量」改为可管理：添加链路(选设备+目标)、删除、单条/全部「立即探测」,展示实测评分/延迟/抖动/丢包/最近探测时间。
  - 模拟器 `rosim` 支持 `/ping`(合成 RTT/丢包);已端到端验证 创建→探测(实测 latency/jitter/loss/score/status)→删除。
- **全功能远程配置(remote configuration of every RouterOS menu)**:
  - `routeros.Catalog`:按 WebFig 菜单分组的配置段目录(Interfaces/WiFi/Wireless/WireGuard/PPP/Bridge/Switch/Mesh/IP/IPv6/MPLS/Routing/System/Queues/Dot1X/RADIUS/SNMP/Tools),覆盖全部菜单;`routeros.ValidPath` 校验任意 RouterOS 资源路径(防注入/穿越)。
  - `routeros.Client.Set`:单例设置走 `POST /<path>/set`(如 /ip/dns、/system/identity);`inventory.REST{List,Create,Update,Delete,Set}`:用托管设备的**保存凭据免登录**远程读写任意配置段。
  - API:`GET /config/catalog`、`GET/POST/PATCH/DELETE /devices/{id}/rest`(通用 CRUD,`singleton:true` 走 set),全部审计。
  - 前端新增**「远程配置」**页:左侧 WebFig 式菜单树 + 右侧配置项表格(过滤/分页列),支持新增/编辑/删除与单例「编辑设置」,通用键值表单即时下发,无需登录设备。
  - 模拟器 `rosim` 支持单例 `/set` 命令端点,便于端到端验证(已验证 identity/dns set + wireguard 等多菜单 CRUD)。

### Fixed
- **DoH 证书校验参数**：之前下发 DoH 时硬编码 `verify-doh-cert=yes`,但 IP 形式的 DoH 端点(如 `https://202.101.51.194:9291/dns-query`)无法校验证书会导致解析失败。现按真实 RouterOS 写法：`dns.BuildConfig` 接受 `Options{VerifyDoHCert}`,nil 时自动判定(IP 端点→`no`,域名端点→`yes`),支持显式覆盖;`POST /devices/{id}/dns` 增 `verify_doh_cert`,前端下发卡片在勾选 DoH 时显示「验证 DoH 证书」开关(自动/关闭/开启)。已端到端验证三种情形。
- **配置备份无法查看**：快照仅存了 `state`(JSON)但无接口返回内容,前端列表只显示「时间 + N 条配置」无法展开。新增 `GET /devices/{id}/snapshots/{snapshotId}`(返回完整 `state.statements`)与 `inventory.GetSnapshot`(设备/租户校验),前端快照行可点击展开,按 RouterOS 段分组显示配置项并支持关键字过滤。已在模拟器上端到端验证(enroll→snapshot→list→查看内容,含 404 校验)。

### Added
- **国内外加速·路由表分流(chnroutes route-table split)**:
  - `chnroutes` 包:从 chnroutes2(默认 `misakaio/chnroutes2`)拉取并缓存国内 IPv4 网段表(解析/去重/校验,支持自定义源 URL)。
  - `accel.BuildChinaSplitScript`:生成 RouterOS `.rsc` 脚本——国内网段(路由表全部条目)走本地 WAN 出口直连,海外 `0.0.0.0/1`+`128.0.0.0/1`(比默认路由更具体,覆盖全网但被国内更具体前缀压制)走 SD-WAN 隧道;脚本幂等(按 neko 注释先清理再下发),支持 `routing-table`/`distance`。
  - `routeros.Client.RunScript`:把上千条路由打包为单个 `/system/script` 一次性安装并执行(免去逐条 REST,免登录设备)。
  - API:`GET /chnroutes`(状态)、`POST /chnroutes/refresh`(刷新)、`POST /devices/{id}/accel/china-split`(`dry_run` 预览脚本/否则下发)。
  - 前端「站点编排」新增**加速·国内外分流(chnroutes)**模式:展示国内路由表条数 + 刷新、填写国内直连出口、预览生成的脚本与条数、一键下发。
- **真正的设备托管(production device management)**:
  - `secret` 包:AES-256-GCM 对设备凭据做静态加密(密钥来自 `NEKO_SECRET_KEY`,部署脚本生成)。
  - 凭据仓储(memory+pg,复用 `device_credentials` 表)+ 设备实时状态(`status` JSONB,迁移 0005)+ `enrolled` 标记。
  - `inventory.Enroll`:录入凭据→加密保存→连接设备(RouterOS REST)→拉取型号/能力→置为 managed;`inventory.Poll`:用保存的凭据刷新在线/版本/CPU/内存/接口实时状态。免登录设备即可管理。
  - API:`POST /devices/{id}/enroll`、`POST /devices/{id}/poll`;设备列表/详情返回实时状态。
  - Worker:每 30s 对已托管设备轮询健康状态。
  - **RouterOS v7 REST 模拟器** `cmd/rosim`(自签 HTTPS):无物理设备也能端到端演示纳管/轮询/下发;compose 增 `rosim` 服务并种子一台 `sim-edge-01` 指向它。
  - 前端:设备列表显示在线/版本/CPU·内存 + 「纳管/轮询」操作;新增**设备详情页**(身份/实时状态/能力矩阵/接口)。
  - 已在真实 PostgreSQL 16 + 模拟器上端到端验证(register→enroll→managed→live poll)。

### Fixed
- **修复运营端登记骨干/出口节点报 500(Postgres)**：平台自营节点(backbone/gateway/POP)不归属租户,但 `devices.tenant_id` 原为 `NOT NULL + 外键`,运营端创建时空 tenant 触发约束错误。新增迁移 `0004_device_tenant_nullable.sql`(tenant_id 可空),Postgres 写入空租户时存 NULL、读取时容错。已在真实 PostgreSQL 16 上复现并验证修复(HTTP 500 → 201)。

### Added
- **站点编排 · 链路选择 · 一键下发(易用性重构第一步)**：新增 `linkpolicy` 包(主备 failover 按优先级生成带 distance + check-gateway 的默认路由,断线自动切换/回切;loadbalance 生成按权重的 ECMP 路由 + NAT)。`configengine.Merge` 支持多策略叠加。新增统一编排接口 `POST /api/v1/devices/{id}/orchestrate`:把链路策略 + 加速模式合成为期望配置,`dry_run` 预览(diff+风险)或经 RouterOS REST 一键下发(commit-confirm/回滚)。前端新增「编排下发」页:选设备→选链路(主备/负载均衡,可增删上行)→可叠加加速→预览生成配置→一键下发并显示结果。把原本分散的引擎串成一个可用工作流。
- **初始化易用性 & Ubuntu 24.04 一键部署**：新增 `scripts/deploy-ubuntu.sh`（`make deploy`）——自动安装 Docker/compose、生成带随机密钥的 `.env`（数据库密码、管理员密码）、按服务器 IP 自动填充对外地址、构建并启动全栈、等待健康检查并打印访问地址与管理员凭据，幂等可重复运行。后端支持 `NEKO_ADMIN_EMAIL/NEKO_ADMIN_PASSWORD` 配置初始运营账号；compose 全面参数化（密码/端口/对外地址经 `.env` 注入）；web 镜像构建期注入 `NEXT_PUBLIC_API_BASE_URL` 以便浏览器正确访问 API。
- **骨干节点管理**：设备新增角色（`cpe`/`backbone`/`gateway`）与地域字段；骨干 POP / 出口网关（均为 RouterOS）统一纳管。迁移 `0003_device_role.sql`；`GET /api/v1/devices?role=` 筛选；前端「骨干节点」页（登记骨干/出口、角色/地域/平台展示）。
- **加速业务·海外运营模式**：`accel` 包新增三种模式——`overseas_direct`（海外运营：全量直连海外出口 IP，**不做分流**，仅默认路由+NAT+海外 DNS）、`smart_split`（智能分流：海外地址表 mangle 标记走隧道、国内直连）、`domestic_direct`。`POST /api/v1/accel/preview` 生成 RouterOS 配置预览 + 风险分级；前端「加速」页可视化选择模式与参数并预览生成配置。
- **ROS 全功能配置（免登录设备）**：`routeros.Client` 完整 REST CRUD（List/Create/Update/Delete 任意配置段）+ `routeros.Applier` 实现 `configengine.Applier`（snapshot→diff→apply→Restore 回滚），通过 REST 直接下发全量配置，无需登录设备控制台；`ManagedSections` 覆盖接口/地址/路由/防火墙/NAT/DHCP/DNS/VLAN/网桥/隧道/队列/SNMP/系统等 26 个配置段。新增 `GET/PUT /api/v1/devices/{id}/config` 与 `GET /api/v1/config/sections`。`ComputeDiff` 现为新增项填充完整属性，使下发计划自包含。
- 任务队列全量收尾（TASKS.md Epic 1–10 全部完成）：
  - 持久化（T1.1/T1.3）：pgx PostgresStore（tenants/devices）+ 嵌入式迁移执行器 + 0002 RLS 行级安全策略（current_tenant() GUC）；`NEKO_STORE=postgres` 启用。
  - 审计（T1.4）：写操作埋点（create/trust_change）+ `/api/v1/audit` 查询。
  - 设备纳管（T2.1）：RouterOS v7 REST 客户端（解析 resource/routerboard/package/license/device-mode/interface，自签 TLS 容错），接入能力检测。
  - 配置引擎（T3.3/3.4/3.5）：Execute 安全下发管线（snapshot→diff→风险闸门→commit-confirm→verify→confirm/restore）+ Canary 灰度分批（1→5%→25%→100%）。
  - 组网（T4.1/4.2）：Overlay 隧道编排（WireGuard/EoIP/GRE 按能力）+ OSPF/重分发语句生成。
  - SNMP（T5.1/5.2/5.4/5.5）：gosnmp v2c/v3 引擎 + 并发网段发现 + Trap 接收器 + 告警 Manager（抑制/升级/Notifier）。
  - DNS（T7.1/7.3）：直连健康检查器 + RouterOS DNS 配置与分流生成。
  - 链路质量（T8.1/8.2）：ICMP/TCP/HTTP/HTTPS/DNS 探测引擎 + 评分 + VictoriaMetrics 上报。
  - 可观测/大盘（T9.1/9.2/9.3）：`/metrics` Prometheus 端点 + 请求埋点 + 真实数据驱动的角色感知大盘 + SVG 拓扑图。
  - 交付（T10.2）：后端 distroless + 前端 Next standalone Dockerfile + `docker-compose.deploy.yml` 全栈编排。
- 登录与真实可用功能：`users` 包（账号 + 盐值迭代 SHA-256 口令哈希，演示账号）、`session` 包（不透明 Bearer Token 会话，含过期，实现 `auth.Authenticator`）、`/api/v1/auth/login`、`/auth/me`、`/auth/logout` 端点；`make demo` 下默认启用鉴权（无 token 返回 401）。前端：`/login` 登录页 + `middleware.ts` 路由保护（未登录跳转登录）+ 会话 Cookie + SSR 携带 Token 鉴权拉取 + 顶栏用户信息与退出登录；**真实创建流程**：新建租户、登记设备（写入后端并刷新）。演示账号：运营 `admin@neko.io / neko12345`，租户 `ops@acme-corp.com / acme12345`。
- 权威开发说明书 `AGENTS.md`，整合全部需求（多租户、设备纳管与能力矩阵、Desired State 配置引擎、SD-WAN 组网与 OSPF/BGP、原生 SNMP、生命周期升级与初始化、DNS 中国区调度、链路质量监控与切换、现代化 Web UI、持续开发机制）。
- 项目文档：`docs/TASKS.md` 任务队列、`docs/DECISIONS.md` ADR、`docs/DESIGN.md` UI 规范、`docs/ARCHITECTURE.md` 架构、`docs/API.md` API 约定。
- 仓库基建：根 `README.md`、`.gitignore`、`Makefile`、`docker-compose.yml`（Postgres/Redis/NATS/VictoriaMetrics/OTel）、`.env.example`、`LICENSE`。
- Go 后端骨架：`config`、结构化日志、HTTP server、健康检查、优雅退出、统一响应 envelope。
- PostgreSQL 迁移脚本：tenants/users/sites/devices/links/audit 等核心表。
- 首个纵向切片：Tenant + Device REST API（内存仓储，可无 DB 运行）+ 单元测试。
- Next.js 14 + TypeScript 控制台骨架与现代化暗色 UI shell（仪表盘 / 设备 / 租户）。
- Demo 与监控读模型：`catalog` 包（链路/告警/DNS 读模型 + 租户作用域列表）；`seed` 包（NEKO_SEED=true 注入 3 租户/5 设备含能力矩阵/5 链路含评分/3 告警/6 DNS）；新增 `/api/v1/links`、`/api/v1/alerts`、`/api/v1/dns/servers` 端点。前端 Links/DNS 页改为实时数据 + 新增 Alerts 页与导航。一键 Demo：`make demo` / `scripts/demo.sh` 启动后端（内存+演示数据）+ 前端控制台，无需外部依赖。
- Epic 1：鉴权与审计基础。`auth` 包：Principal（运营/租户作用域）、Token SHA-256 哈希存储、内存 Authenticator；httpapi 可选 Bearer Token 鉴权中间件（NEKO_AUTH=on 启用，运营 token 可经 X-Tenant-Id 切租户，租户 token 锁定自身，健康端点公开）。`audit` 包：append-only 审计记录器（记录 who/when/object/before/after，仅追加与查询）。均覆盖单元测试。
- Epic 6：生命周期管理。`lifecycle` 包：RouterOS 版本解析与数值比较、`NeedsUpgrade`；`PlanUpgrade` 生成受控升级步骤（下载→校验→升级→重启→验证→健康检查；RouterBOARD 额外的 RouterBOOT 升级排在其后并二次重启，CHR/x86 跳过）；`InitTemplate` 生成设备开局 Desired State（identity/时区/NTP/SNMP/管理防火墙）。覆盖单元测试。
- Epic 4：SD-WAN 动态路由。`routing` 包：路由意图模型（静态/OSPF/BGP/汇总/重分发 + 每租户 VRF 与 community），BGP 邻居 eBGP/iBGP 自动分类，`Validate` 防路由泄漏校验（VRF/community 必填、eBGP 强制 import+export 过滤、重分发强制过滤、iBGP 全互联建议 RR、eBGP 建议 BFD），`BuildState` 将意图生成 RouterOS v7 声明式路由语句（含 RR client、BFD、VRF 隔离）。覆盖单元测试。
- Epic 5：SNMP 指标计算与告警引擎。`snmp` 包：接口计数器→octet 速率（32 位回绕处理、64 位重置识别）、bps 转换、接口利用率。`alerting` 包：阈值规则引擎，支持比较操作符、for-duration 持续确认、firing/resolved 状态转换、去重（稳态不重复告警）、多 series 隔离。均覆盖单元测试。
- Epic 7：DNS 中国区调度。`dns` 包：DNS 服务器池模型（运营商 telecom/unicom/mobile/edu/public、地域、ECS、健康、延迟），`Select` 按运营商匹配/地域匹配/ECS/延迟加权评分并公共 anycast 兜底，输出确定性 best-first 列表。覆盖单元测试。
- Epic 8：链路质量评分与切换。`linkqos` 包：`Score` 按延迟/丢包/抖动加权计算 0..100 评分；`Controller` 实现带防震荡的切换状态机（滞后阈值 UpScore/DownScore、MinDown/MinUp 持续时间确认、MinDwell 最小驻留窗口），支持主备角色优先、故障切换与自动回切。覆盖单元测试（评分边界/初始选主/故障切换/驻留抑制/自动回切）。
- Epic 3：Desired State 配置引擎核心。`configengine` 包：声明式配置模型（Statement/State），`ComputeDiff` 计算属性级最小变更集（确定性排序），风险分级（按 RouterOS 路径基线 + 删除操作升级 + 管理接口/地址变更升级为 critical 以保护管理通道）。覆盖单元测试（增改删/无变更/风险聚合/管理通道/删除升级）。
- Epic 2：RouterOS 设备能力识别。`routeros` 包：DeviceFacts 模型 + `Detect`（识别 RouterBOARD/CHR/x86、版本、架构、软件包、License、Device Mode、接口能力，归一化为能力矩阵；基于能力而非型号字符串）+ Collector 接口与 StaticCollector。`inventory` 包新增 Trust State 状态机、`Detect`（探测并丰富设备 + 推进信任状态）、`SetTrustState`，以及 `/devices/{id}/detect`、`/devices/{id}/trust` API 端点。覆盖单元测试。
