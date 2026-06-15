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
| GET | `/api/v1/tenants?include=stats` | 租户列表（运营端，含设备/站点/告警统计） |
| POST | `/api/v1/tenants` | 创建租户 |
| GET | `/api/v1/tenants/{id}?include=stats` | 租户详情 |
| PATCH | `/api/v1/tenants/{id}` | 更新租户（名称 / slug / 状态 active|suspended） |
| DELETE | `/api/v1/tenants/{id}` | 删除租户（body: `{confirm_slug}`，级联删除关联数据） |
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
| GET | `/api/v1/devices/{id}/snapshots/{snapshotId}` | 查看单个配置快照内容（返回 `state.statements` 完整配置项） |
| POST | `/api/v1/devices/{id}/snapshots/{snapshotId}/restore` | 将设备配置还原到指定快照（托管凭据，diff→apply） |
| POST | `/api/v1/devices/{id}/snapshots/{snapshotId}/apply` | 下发编辑后的配置 `{state:{statements:[]}}` 到设备 |
| POST | `/api/v1/devices/{id}/orchestrate` | 站点编排：链路选择(failover/ECMP)+加速 合成配置，`dry_run` 预览或一键下发 |
| GET | `/api/v1/accel/modes` | 加速业务模式列表（含海外直连 overseas_direct） |
| POST | `/api/v1/accel/preview` | 生成加速模式对应的 RouterOS 配置预览 |
| POST | `/api/v1/accel/propose` | CPE→POP 加速：自动生成 WG 密钥/overlay 地址/隧道参数 + CPE/POP 双侧配置预览 |
| POST | `/api/v1/fabric/deploy` | 生产级双向下发：POP 先、CPE 后，WireGuard 隧道 + 加速/组网路由，使用托管凭据。`dry_run` 预览 |
| POST | `/api/v1/mesh/deploy` | 多站点组网：`topology`=`hub_spoke`\|`transit`\|`full_mesh`，`sites[]`（CPE+POP+prefixes），`backbone_path`（transit/full_mesh），BGP AS + 骨干间 WG/iBGP |
| GET | `/api/v1/qos/policies` | 限速策略池（RouterOS Simple Queue 模板） |
| POST | `/api/v1/qos/policies` | 添加快限速策略 `{name,target,max_limit,limit_at?,priority?}` |
| DELETE | `/api/v1/qos/policies/{id}` | 删除限速策略 |
| POST | `/api/v1/devices/{id}/qos` | 下发 Simple Queue 到设备，`policy_ids` 或内联 `rules`，支持 `dry_run` |
| GET | `/api/v1/links` | 监控链路列表（设备实测的延迟/抖动/丢包/评分） |
| POST | `/api/v1/links` | 新增监控链路 `{device_id, name, kind, isp, role, target}` |
| DELETE | `/api/v1/links/{id}` | 删除监控链路 |
| POST | `/api/v1/links/{id}/probe` | 即时探测：从设备 Ping 目标并更新质量 |
| GET | `/api/v1/config/sections` | 平台全功能管理的 RouterOS 配置段目录 |
| GET | `/api/v1/config/catalog` | 按 WebFig 菜单分组的全部可远程配置段（菜单树） |
| GET | `/api/v1/devices/{id}/rest?path=/ip/address` | 远程读取任意 RouterOS 配置段（用托管凭据，免登录） |
| POST | `/api/v1/devices/{id}/rest` | 远程新增配置项 `{path, attributes}`；`singleton:true` 走 set（如 /ip/dns） |
| PATCH | `/api/v1/devices/{id}/rest` | 远程修改配置项 `{path, item_id, attributes}` |
| DELETE | `/api/v1/devices/{id}/rest?path=&item_id=` | 远程删除配置项 |
| GET | `/api/v1/chnroutes` | 国内路由表(chnroutes2)缓存状态（条数/来源/更新时间；`sources` 为候选 URL 列表） |
| POST | `/api/v1/chnroutes/refresh` | 刷新国内路由表，可选 `{url}`；未指定时按顺序尝试 jsdelivr → ghproxy → GitHub raw，可通过 `NEKO_CHNROUTES_URL` / `NEKO_CHNROUTES_URLS` 覆盖 |
| POST | `/api/v1/devices/{id}/accel/china-split` | 国内外分流下发：国内走路由表本地直连，海外 0.0.0.0/1+128.0.0.0/1 走隧道。`dry_run` 预览生成的 RouterOS 脚本，否则一键安装+执行 |

> 新增端点请同步更新本表与 `web/lib/api.ts` 类型。
