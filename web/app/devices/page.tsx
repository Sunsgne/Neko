import { Card } from "@/components/ui";
import { listDevices, type Device } from "@/lib/api";
import { serverToken } from "@/lib/server-session";
import { RegisterDeviceButton } from "@/components/register-device";
import { DeviceTable } from "@/components/device-table";

export const dynamic = "force-dynamic";

export default async function DevicesPage() {
  let devices: Device[] = [];
  try {
    devices = await listDevices(serverToken());
  } catch {
    devices = [];
  }

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">设备纳管</h1>
          <p className="mt-1 text-sm text-muted">
            自动识别 RouterBOARD / CHR / x86，基于能力下发配置；支持单台或批量删除
          </p>
        </div>
        <RegisterDeviceButton />
      </div>

      <Card className="p-0">
        {devices.length === 0 ? (
          <div className="flex flex-col items-center justify-center gap-2 py-16 text-center">
            <p className="text-sm text-muted">还没有设备</p>
            <p className="text-xs text-muted">点击右上角「登记设备」或在「发现纳管」中扫描网段批量添加</p>
          </div>
        ) : (
          <DeviceTable devices={devices} />
        )}
      </Card>
    </div>
  );
}
