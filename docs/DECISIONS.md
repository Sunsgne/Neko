# DECISIONS.md — 架构决策记录 (ADR)

> 每条记录：背景 / 决策 / 理由 / 影响。新决策追加在末尾。

## ADR-0001：后端选用 Go
- **背景**：需要大规模并发编排 RouterOS 设备（SSH/API）、SNMP 轮询、主动探测、DNS 调度。
- **决策**：后端控制平面与所有 worker 使用 Go 1.22+。
- **理由**：goroutine 并发模型契合海量并发 I/O；静态编译单二进制部署简单；网络生态成熟（gosnmp、crypto/ssh、pgx）；GC 延迟低，适合常驻服务。
- **影响**：团队需具备 Go 能力；前后端语言不同，通过 OpenAPI/类型生成对齐。

## ADR-0002：前端选用 Next.js + TypeScript
- **背景**：需要现代化、高端的运营/租户双端控制台。
- **决策**：Next.js 14 App Router + TypeScript + Tailwind + shadcn/ui。
- **理由**：RSC/SSR 性能好，DX 优秀，UI 生态一流，便于做专业控制台。
- **影响**：与 Go 后端通过 REST + 类型契约协作。

## ADR-0003：数据与消息基础设施
- **决策**：PostgreSQL（核心数据 + RLS 多租户）、Redis（缓存/锁/限流）、NATS JetStream（事件/任务/上报）、VictoriaMetrics（时序）、OpenTelemetry（可观测）。
- **理由**：各组件在各自领域成熟且运维成本可控；NATS JetStream 轻量且适合设备上报与任务分发；VictoriaMetrics 资源占用低、查询快。
- **影响**：本地通过 docker-compose 一键拉起。

## ADR-0004：仓储接口先行，内存实现打底
- **背景**：希望项目从第一天起即可运行、可测，不被 DB 连接阻塞开发。
- **决策**：定义 `store` 仓储接口，Bootstrap 阶段提供内存实现；Epic 1 再切换 pgx 实现。
- **理由**：降低初期摩擦，保证纵向切片可运行；接口隔离便于后续替换与测试。
- **影响**：需保证内存实现与 pg 实现行为一致（共享契约测试）。

## ADR-0005：配置下发采用 Desired State + Diff + 安全回滚
- **背景**：绝不能按型号下发固定脚本，且不能切断管理通道。
- **决策**：声明式期望状态 → 抓取 running → 计算 diff → 风险分级 → commit-confirm/safe-mode 下发 → 自动验证 → 失败回滚 → Canary 灰度。
- **理由**：可预测、可审计、可回滚，保护管理通道，适配异构设备能力。
- **影响**：实现复杂度上移到平台，设备侧保持轻量。
