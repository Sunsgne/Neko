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

> 新增端点请同步更新本表与 `web/lib/api.ts` 类型。
