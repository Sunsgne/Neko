-- 0001_init.sql — Neko 平台核心 schema
-- 多租户：每张业务表带 tenant_id；Epic 1 (T1.3) 再启用 PostgreSQL RLS。
-- 迁移仅追加（forward-only）。

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ── 租户 ───────────────────────────────────────────────
CREATE TABLE tenants (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    status      TEXT NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active','suspended')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ── 用户与 RBAC ────────────────────────────────────────
CREATE TABLE users (
    id            TEXT PRIMARY KEY,
    tenant_id     TEXT REFERENCES tenants(id) ON DELETE CASCADE, -- NULL = 平台运营用户
    email         TEXT NOT NULL,
    display_name  TEXT NOT NULL DEFAULT '',
    password_hash TEXT NOT NULL DEFAULT '',
    is_operator   BOOLEAN NOT NULL DEFAULT false,
    status        TEXT NOT NULL DEFAULT 'active',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, email)
);

CREATE TABLE roles (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT REFERENCES tenants(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    permissions JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE user_roles (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE api_tokens (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT REFERENCES tenants(id) ON DELETE CASCADE,
    user_id     TEXT REFERENCES users(id) ON DELETE SET NULL,
    name        TEXT NOT NULL,
    token_hash  TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ
);

-- ── 站点 ───────────────────────────────────────────────
CREATE TABLE sites (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    region      TEXT NOT NULL DEFAULT '',
    isp         TEXT NOT NULL DEFAULT '',  -- 运营商：telecom/unicom/mobile/edu
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ── 设备（RouterOS）+ 能力矩阵 ─────────────────────────
CREATE TABLE devices (
    id           TEXT PRIMARY KEY,
    tenant_id    TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    site_id      TEXT REFERENCES sites(id) ON DELETE SET NULL,
    name         TEXT NOT NULL,
    mgmt_address TEXT NOT NULL,
    platform     TEXT NOT NULL DEFAULT 'unknown'
                   CHECK (platform IN ('routerboard','chr','x86','unknown')),
    model        TEXT NOT NULL DEFAULT '',
    serial       TEXT NOT NULL DEFAULT '',
    trust_state  TEXT NOT NULL DEFAULT 'discovered'
                   CHECK (trust_state IN ('untrusted','discovered','authenticated','enrolled','managed')),
    capabilities JSONB,        -- 归一化能力矩阵（见 store.CapabilityMatrix）
    last_seen_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_devices_tenant ON devices(tenant_id);
CREATE INDEX idx_devices_trust  ON devices(trust_state);

-- 加密存储的设备凭据（密钥经 KMS/age 加密后入库，不存明文）
CREATE TABLE device_credentials (
    device_id      TEXT PRIMARY KEY REFERENCES devices(id) ON DELETE CASCADE,
    kind           TEXT NOT NULL,            -- ssh-key | api | password
    encrypted_blob BYTEA NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ── 链路（WAN/Overlay）+ 质量 ─────────────────────────
CREATE TABLE links (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    device_id   TEXT REFERENCES devices(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    kind        TEXT NOT NULL DEFAULT 'wan'  -- wan | overlay
                   CHECK (kind IN ('wan','overlay')),
    isp         TEXT NOT NULL DEFAULT '',
    role        TEXT NOT NULL DEFAULT 'primary'  -- primary | backup
                   CHECK (role IN ('primary','backup')),
    status      TEXT NOT NULL DEFAULT 'unknown',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_links_tenant ON links(tenant_id);

-- ── 配置引擎：期望状态快照 ─────────────────────────────
CREATE TABLE config_desired (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    device_id   TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    version     INTEGER NOT NULL,
    spec        JSONB NOT NULL,
    created_by  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (device_id, version)
);

CREATE TABLE config_runs (
    id           TEXT PRIMARY KEY,
    tenant_id    TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    device_id    TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    desired_id   TEXT REFERENCES config_desired(id) ON DELETE SET NULL,
    diff         JSONB,
    risk         TEXT NOT NULL DEFAULT 'low'
                   CHECK (risk IN ('low','medium','high','critical')),
    status       TEXT NOT NULL DEFAULT 'planned'
                   CHECK (status IN ('planned','applying','verifying','committed','rolledback','failed')),
    started_at   TIMESTAMPTZ,
    finished_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_config_runs_device ON config_runs(device_id);

-- ── DNS 服务器池 ───────────────────────────────────────
CREATE TABLE dns_servers (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT REFERENCES tenants(id) ON DELETE CASCADE,
    address     TEXT NOT NULL,
    region      TEXT NOT NULL DEFAULT '',
    isp         TEXT NOT NULL DEFAULT '',
    supports_ecs BOOLEAN NOT NULL DEFAULT false,
    healthy     BOOLEAN NOT NULL DEFAULT true,
    latency_ms  INTEGER,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ── 告警 ───────────────────────────────────────────────
CREATE TABLE alerts (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    device_id   TEXT REFERENCES devices(id) ON DELETE SET NULL,
    severity    TEXT NOT NULL DEFAULT 'warning'
                   CHECK (severity IN ('info','warning','critical')),
    title       TEXT NOT NULL,
    detail      TEXT NOT NULL DEFAULT '',
    state       TEXT NOT NULL DEFAULT 'firing'
                   CHECK (state IN ('firing','resolved')),
    fired_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ
);
CREATE INDEX idx_alerts_tenant_state ON alerts(tenant_id, state);

-- ── 审计日志（追加写，不可篡改约定）────────────────────
CREATE TABLE audit_logs (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT,
    actor_id    TEXT,
    action      TEXT NOT NULL,
    object_type TEXT NOT NULL,
    object_id   TEXT,
    before      JSONB,
    after       JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_tenant_time ON audit_logs(tenant_id, created_at DESC);
