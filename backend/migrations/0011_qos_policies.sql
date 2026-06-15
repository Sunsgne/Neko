-- QoS / rate-limit policy pool (maps to RouterOS /queue/simple).
CREATE TABLE IF NOT EXISTS qos_policies (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT REFERENCES tenants(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    target      TEXT NOT NULL,
    max_limit   TEXT NOT NULL,
    limit_at    TEXT NOT NULL DEFAULT '',
    burst_limit TEXT NOT NULL DEFAULT '',
    priority    INT  NOT NULL DEFAULT 8,
    comment     TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_qos_policies_tenant ON qos_policies(tenant_id);
