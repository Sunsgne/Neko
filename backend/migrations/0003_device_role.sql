-- 0003_device_role.sql — 设备角色（骨干节点 / CPE / 网关）与地域
-- 骨干节点（POP）也是 RouterOS，纳入统一设备纳管。

ALTER TABLE devices ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'cpe'
    CHECK (role IN ('cpe','backbone','gateway'));
ALTER TABLE devices ADD COLUMN IF NOT EXISTS region TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_devices_role ON devices(role);
