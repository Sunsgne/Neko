# Migrations

仅追加（forward-only）SQL 迁移。命名：`NNNN_description.sql`。

Bootstrap 阶段 API 默认使用内存仓储（`NEKO_STORE=memory`），无需数据库即可运行。
Epic 1（T1.1）引入 pgx 与迁移执行器后，设置 `NEKO_STORE=postgres` 并配置 `DATABASE_URL` 即可切换。

本地应用迁移（任选其一）：

```bash
# 使用 psql 直接执行
psql "$DATABASE_URL" -f backend/migrations/0001_init.sql

# 或 docker compose 起 postgres 后
docker compose up -d postgres
psql postgres://neko:neko@localhost:5432/neko -f backend/migrations/0001_init.sql
```
