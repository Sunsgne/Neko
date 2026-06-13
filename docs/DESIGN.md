# DESIGN.md — Web 控制台 UI/UX 详规

## 设计原则
- **专业控制台优先**：暗色为主，数据密度高，层级清晰，状态色明确。
- **克制的现代感**：圆角、柔和阴影、细腻分隔线；动效短促有目的（150–250ms）。
- **一致性**：所有组件来自 shadcn/ui，统一 spacing / radius / typography token。

## 色彩 Token（dark-first）
| 用途 | 值 (HSL) |
| --- | --- |
| background | `222 47% 7%` |
| surface / card | `222 40% 11%` |
| border | `222 25% 20%` |
| foreground | `210 40% 96%` |
| muted-foreground | `215 20% 65%` |
| primary (品牌蓝) | `199 89% 52%` |
| success | `152 60% 45%` |
| warning | `38 92% 55%` |
| danger | `0 72% 55%` |

状态语义：success=链路健康/已收敛；warning=告警/降级；danger=故障/中断；primary=信息/操作。

## 排版
- 字体：`Inter`（UI）+ `JetBrains Mono`（IP/配置/日志等等宽内容）。
- 标题 `text-2xl/3xl font-semibold tracking-tight`；正文 `text-sm`；标签 `text-xs uppercase tracking-wide muted`。

## 布局
- **AppShell**：左侧可折叠侧边栏（图标+文字）+ 顶栏（租户切换、搜索 ⌘K、通知、用户菜单）+ 主内容区。
- 主内容：卡片网格（KPI 卡 → 图表 → 表格）。
- 间距基准 4px；卡片 `rounded-xl border bg-card p-5`。

## 关键页面
1. **Dashboard**：KPI（设备总数/在线率/活跃告警/总流量）、流量趋势图、链路质量热力、最近事件。
2. **Devices**：表格（型号/版本/平台/Trust State/状态），抽屉详情含能力矩阵。
3. **Tenants**（运营端）：租户列表、用量、状态。
4. **Topology**：React Flow 站点/POP/链路拓扑，链路按评分着色。
5. **Routing / DNS / Links / Alerts**：各功能域管理页。

## 组件库与依赖
- Tailwind CSS、shadcn/ui (Radix)、lucide-react、Recharts、TanStack Query、TanStack Table、React Flow。

## 交互细则
- 加载用骨架屏；写操作乐观更新；实时数据走 SSE/WebSocket。
- 空态有插画/引导；错误态有重试；危险操作二次确认 + 风险标识。
- 全键盘可达，⌘K 命令面板可跳转/执行常用操作。
