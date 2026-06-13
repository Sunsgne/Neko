# ARCHITECTURE.md — 架构细节

## 分层
- **cmd/**：可执行入口（`api`、`worker`），仅做装配（wire），不含业务逻辑。
- **internal/<domain>/**：每个功能域自包含 `service`（业务）+ `model`（领域模型）+（必要时）`handler`。
- **internal/store/**：仓储接口 + 内存实现 + pgx 实现 + 迁移。
- **internal/httpapi/**：路由、中间件（请求 ID、日志、恢复、CORS、鉴权、租户作用域）、统一响应/错误 envelope。
- **internal/observability/**：slog 结构化日志 + OpenTelemetry 初始化。
- **internal/config/**：环境变量加载与校验。

## 依赖方向
`cmd` → `httpapi` → `domain service` → `store interface`。domain 不依赖 httpapi；store 实现细节不外泄。

## 数据流（示例：设备纳管）
1. 运营/租户在 Web 发起纳管 → API 创建 device(record, state=discovered)。
2. 发布 `device.onboard` 事件到 NATS。
3. Inventory Worker 消费：连 RouterOS 读取型号/版本/能力 → 写能力矩阵 → state=authenticated/enrolled。
4. 结果事件回传，前端经 SSE 实时刷新。

## 多租户
- 每张业务表含 `tenant_id`；运营端可跨租户（特权角色）。
- DB 层 PostgreSQL RLS：会话设置 `app.tenant_id`，策略强制隔离。
- API 中间件解析 token → 注入 tenant 作用域 → 传递到 store。

## 配置引擎状态机
`desired(version N)` + `running(snapshot)` → `plan(diff, risk)` → `apply(canary, commit-confirm)` → `verify` → `committed | rolledback`。

## 事件主题（NATS）
- `device.onboard` / `device.config.apply` / `device.config.result`
- `snmp.poll` / `snmp.trap`
- `probe.report` / `link.switch`
- `lifecycle.upgrade` / `lifecycle.init`

## 指标（VictoriaMetrics，命名约定）
- `neko_iface_in_bps{tenant,device,iface}` / `neko_iface_out_bps`
- `neko_link_latency_ms{tenant,link}` / `neko_link_loss_ratio` / `neko_link_jitter_ms` / `neko_link_score`
- `neko_device_cpu_ratio` / `neko_device_mem_ratio` / `neko_device_temp_c`

## 可观测性
- 所有服务初始化 OTel TracerProvider/MeterProvider，OTLP 导出到 collector。
- 日志带 trace_id；HTTP 中间件创建 root span。
