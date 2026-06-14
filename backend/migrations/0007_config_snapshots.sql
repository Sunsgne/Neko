-- 0007_config_snapshots.sql — 设备配置快照（备份历史 + 漂移检测）
CREATE TABLE IF NOT EXISTS config_snapshots (
    id              TEXT PRIMARY KEY,
    tenant_id       TEXT REFERENCES tenants(id) ON DELETE CASCADE,
    device_id       TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    source          TEXT NOT NULL DEFAULT 'manual',
    state           JSONB NOT NULL,
    statement_count INTEGER NOT NULL DEFAULT 0,
    taken_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_snapshots_device_time ON config_snapshots(device_id, taken_at DESC);
