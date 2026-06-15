-- 0010_links.sql — 监控链路（WAN/overlay）及其最新质量快照
CREATE TABLE IF NOT EXISTS links (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT REFERENCES tenants(id) ON DELETE CASCADE,
    device_id   TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    kind        TEXT NOT NULL DEFAULT 'wan',
    isp         TEXT NOT NULL DEFAULT '',
    role        TEXT NOT NULL DEFAULT 'primary',
    target      TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'unknown',
    latency_ms  DOUBLE PRECISION NOT NULL DEFAULT 0,
    jitter_ms   DOUBLE PRECISION NOT NULL DEFAULT 0,
    loss        DOUBLE PRECISION NOT NULL DEFAULT 0,
    score       DOUBLE PRECISION NOT NULL DEFAULT 0,
    measured_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_links_device ON links(device_id);
CREATE INDEX IF NOT EXISTS idx_links_tenant ON links(tenant_id);
