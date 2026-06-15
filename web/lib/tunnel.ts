/** Shared overlay / WireGuard helpers for CPE↔POP orchestration UI. */

export function hostFromMgmt(addr: string): string {
  const i = addr.lastIndexOf(":");
  if (i > 0 && addr.includes(".") && !addr.slice(i).includes("]")) {
    return addr.slice(0, i);
  }
  return addr;
}

export function popPeerOf(cpeOverlay: string): string {
  const ip = cpeOverlay.split("/")[0];
  const parts = ip.split(".");
  if (parts.length === 4) {
    parts[3] = "1";
    return parts.join(".");
  }
  return ip;
}

export function tunnelNameForPop(popName: string): string {
  const name = `wg-${popName}`.replace(/[^a-zA-Z0-9-]/g, "-").slice(0, 24);
  return name || "wg-pop";
}
