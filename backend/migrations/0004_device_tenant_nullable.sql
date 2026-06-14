-- 0004_device_tenant_nullable.sql
-- 平台自营节点（骨干 POP / 出口网关）不归属任何租户，tenant_id 允许为空。
-- 运营端创建此类设备时 tenant_id 为 NULL，避免 NOT NULL/外键约束导致 500。

ALTER TABLE devices ALTER COLUMN tenant_id DROP NOT NULL;
