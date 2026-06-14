"use client";

// Client-side session helpers. The token is stored in a cookie (readable by
// server components for authenticated SSR fetches). For the demo this is a
// non-httpOnly cookie; production should issue an httpOnly, Secure cookie.

export { TOKEN_COOKIE, EMAIL_COOKIE, ROLE_COOKIE } from "@/lib/cookies";
import { TOKEN_COOKIE, EMAIL_COOKIE, ROLE_COOKIE } from "@/lib/cookies";

function setCookie(name: string, value: string, maxAgeSec: number) {
  document.cookie = `${name}=${encodeURIComponent(value)}; path=/; max-age=${maxAgeSec}; samesite=lax`;
}

function delCookie(name: string) {
  document.cookie = `${name}=; path=/; max-age=0`;
}

export function getCookie(name: string): string | undefined {
  if (typeof document === "undefined") return undefined;
  const match = document.cookie.split("; ").find((c) => c.startsWith(`${name}=`));
  return match ? decodeURIComponent(match.split("=").slice(1).join("=")) : undefined;
}

export function saveSession(token: string, email: string, isOperator: boolean) {
  const day = 60 * 60 * 12;
  setCookie(TOKEN_COOKIE, token, day);
  setCookie(EMAIL_COOKIE, email, day);
  setCookie(ROLE_COOKIE, isOperator ? "operator" : "tenant", day);
}

export function clearSession() {
  delCookie(TOKEN_COOKIE);
  delCookie(EMAIL_COOKIE);
  delCookie(ROLE_COOKIE);
}

export function currentToken(): string | undefined {
  return getCookie(TOKEN_COOKIE);
}
