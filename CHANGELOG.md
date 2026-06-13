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
