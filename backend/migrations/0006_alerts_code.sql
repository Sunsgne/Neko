-- 0006_alerts_code.sql — 告警去重所需的 code 列
-- monitoring 以 (device_id, code) 为键去重：同一规则在 firing 期间不重复创建。

ALTER TABLE alerts ADD COLUMN IF NOT EXISTS code TEXT NOT NULL DEFAULT '';

-- 同一设备+规则同时只允许一条 firing 告警。
CREATE UNIQUE INDEX IF NOT EXISTS uq_alerts_open
    ON alerts (device_id, code)
    WHERE state = 'firing';
