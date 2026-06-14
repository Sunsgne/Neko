-- 0005_device_status.sql — 设备实时状态与托管标记
-- status: 轮询采集的在线/版本/资源等(JSONB)；enrolled: 是否已存储凭据被平台托管。

ALTER TABLE devices ADD COLUMN IF NOT EXISTS status   JSONB;
ALTER TABLE devices ADD COLUMN IF NOT EXISTS enrolled BOOLEAN NOT NULL DEFAULT false;
