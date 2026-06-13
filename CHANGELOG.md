# Changelog

本项目遵循 [Keep a Changelog](https://keepachangelog.com/) 与 [Conventional Commits](https://www.conventionalcommits.org/)。

## [Unreleased]

### Added
- 权威开发说明书 `AGENTS.md`，整合全部需求（多租户、设备纳管与能力矩阵、Desired State 配置引擎、SD-WAN 组网与 OSPF/BGP、原生 SNMP、生命周期升级与初始化、DNS 中国区调度、链路质量监控与切换、现代化 Web UI、持续开发机制）。
- 项目文档：`docs/TASKS.md` 任务队列、`docs/DECISIONS.md` ADR、`docs/DESIGN.md` UI 规范、`docs/ARCHITECTURE.md` 架构、`docs/API.md` API 约定。
- 仓库基建：根 `README.md`、`.gitignore`、`Makefile`、`docker-compose.yml`（Postgres/Redis/NATS/VictoriaMetrics/OTel）、`.env.example`、`LICENSE`。
- Go 后端骨架：`config`、结构化日志、HTTP server、健康检查、优雅退出、统一响应 envelope。
- PostgreSQL 迁移脚本：tenants/users/sites/devices/links/audit 等核心表。
- 首个纵向切片：Tenant + Device REST API（内存仓储，可无 DB 运行）+ 单元测试。
- Next.js 14 + TypeScript 控制台骨架与现代化暗色 UI shell（仪表盘 / 设备 / 租户）。
- Epic 5：SNMP 指标计算与告警引擎。`snmp` 包：接口计数器→octet 速率（32 位回绕处理、64 位重置识别）、bps 转换、接口利用率。`alerting` 包：阈值规则引擎，支持比较操作符、for-duration 持续确认、firing/resolved 状态转换、去重（稳态不重复告警）、多 series 隔离。均覆盖单元测试。
- Epic 7：DNS 中国区调度。`dns` 包：DNS 服务器池模型（运营商 telecom/unicom/mobile/edu/public、地域、ECS、健康、延迟），`Select` 按运营商匹配/地域匹配/ECS/延迟加权评分并公共 anycast 兜底，输出确定性 best-first 列表。覆盖单元测试。
- Epic 8：链路质量评分与切换。`linkqos` 包：`Score` 按延迟/丢包/抖动加权计算 0..100 评分；`Controller` 实现带防震荡的切换状态机（滞后阈值 UpScore/DownScore、MinDown/MinUp 持续时间确认、MinDwell 最小驻留窗口），支持主备角色优先、故障切换与自动回切。覆盖单元测试（评分边界/初始选主/故障切换/驻留抑制/自动回切）。
- Epic 3：Desired State 配置引擎核心。`configengine` 包：声明式配置模型（Statement/State），`ComputeDiff` 计算属性级最小变更集（确定性排序），风险分级（按 RouterOS 路径基线 + 删除操作升级 + 管理接口/地址变更升级为 critical 以保护管理通道）。覆盖单元测试（增改删/无变更/风险聚合/管理通道/删除升级）。
- Epic 2：RouterOS 设备能力识别。`routeros` 包：DeviceFacts 模型 + `Detect`（识别 RouterBOARD/CHR/x86、版本、架构、软件包、License、Device Mode、接口能力，归一化为能力矩阵；基于能力而非型号字符串）+ Collector 接口与 StaticCollector。`inventory` 包新增 Trust State 状态机、`Detect`（探测并丰富设备 + 推进信任状态）、`SetTrustState`，以及 `/devices/{id}/detect`、`/devices/{id}/trust` API 端点。覆盖单元测试。
