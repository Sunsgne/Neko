# API.md — API 约定

## 基本约定
- 基址：`/api/v1`
- 内容类型：`application/json`
- 鉴权：`Authorization: Bearer <token>`（Bootstrap 阶段可放行只读端点）
- 租户作用域：运营端 token 可跨租户；租户 token 自动限定 `tenant_id`。

## 统一响应 Envelope
成功：
```json
{ "data": <payload>, "meta": { "page": 1, "page_size": 20, "total": 42 } }
```
错误：
```json
{ "error": { "code": "not_found", "message": "device not found", "details": {} } }
```

## 分页
`GET /resource?page=1&page_size=20`，`meta.total` 返回总数。

## 健康检查
- `GET /healthz` — 存活
- `GET /readyz` — 就绪（依赖可用）

## 当前已实现端点（随开发增长）
| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/healthz` | liveness |
| GET | `/readyz` | readiness |
| GET | `/api/v1/tenants` | 租户列表（运营端） |
| POST | `/api/v1/tenants` | 创建租户 |
| GET | `/api/v1/tenants/{id}` | 租户详情 |
| GET | `/api/v1/devices` | 设备列表 |
| POST | `/api/v1/devices` | 登记设备（进入纳管流程） |
| GET | `/api/v1/devices/{id}` | 设备详情（含能力矩阵） |
| POST | `/api/v1/devices/{id}/detect` | 探测并识别型号/能力，推进信任状态 |
| POST | `/api/v1/devices/{id}/trust` | 变更设备信任状态（状态机校验） |
| POST | `/api/v1/tools/config-diff` | 计算 Desired State Diff + 风险分级 |
| POST | `/api/v1/tools/routing/validate` | 路由意图防泄漏校验 + 生成配置 |
| POST | `/api/v1/tools/link-score` | 计算链路质量评分 |
| GET | `/api/v1/devices?role=backbone\|gateway\|cpe` | 按角色筛选设备（骨干节点管理） |
| GET | `/api/v1/devices/{id}/config` | 读取设备实时配置快照（REST，无需登录设备） |
| PUT | `/api/v1/devices/{id}/config` | 下发全功能配置到设备（REST，snapshot→diff→apply→回滚） |
| GET | `/api/v1/accel/modes` | 加速业务模式列表（含海外直连 overseas_direct） |
| POST | `/api/v1/accel/preview` | 生成加速模式对应的 RouterOS 配置预览 |
| GET | `/api/v1/config/sections` | 平台全功能管理的 RouterOS 配置段目录 |

> 新增端点请同步更新本表与 `web/lib/api.ts` 类型。
