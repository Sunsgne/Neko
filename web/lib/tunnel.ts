/** Shared overlay / WireGuard helpers for CPE↔POP orchestration UI. */

export function hostFromMgmt(addr: string): string {
  const i = addr.lastIndexOf(":");
  if (i > 0 && addr.includes(".") && !addr.slice(i).includes("]")) {
    return addr.slice(0, i);
  }
  return addr;
}

export function popPeerOf(cpeOverlay: string): string {
  const parts = cpeOverlay.split("/");
  const ip = parts[0];
  const segs = ip.split(".").map(Number);
  if (segs.length !== 4 || parts[1] !== "30") return ip;
  const base = segs[3] & ~3;
  const host1 = base + 1;
  const host2 = base + 2;
  const last = segs[3];
  if (last === host1) return `${segs[0]}.${segs[1]}.${segs[2]}.${host2}`;
  if (last === host2) return `${segs[0]}.${segs[1]}.${segs[2]}.${host1}`;
  return ip;
}

export function tunnelNameForPop(popName: string): string {
  const name = `wg-${popName}`.replace(/[^a-zA-Z0-9-]/g, "-").slice(0, 24);
  return name || "wg-pop";
}
