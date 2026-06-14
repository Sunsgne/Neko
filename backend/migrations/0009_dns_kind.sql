-- 0009_dns_kind.sql — DNS 服务器类型（普通 UDP / DoH）
ALTER TABLE dns_servers ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'udp'
    CHECK (kind IN ('udp','doh'));
