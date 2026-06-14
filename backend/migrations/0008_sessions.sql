-- 0008_sessions.sql — 持久化登录会话（API 重启后 token 仍有效）
CREATE TABLE IF NOT EXISTS sessions (
    token       TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL,
    email       TEXT NOT NULL DEFAULT '',
    tenant_id   TEXT,
    is_operator BOOLEAN NOT NULL DEFAULT false,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);
