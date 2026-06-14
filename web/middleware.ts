import { NextRequest, NextResponse } from "next/server";

const TOKEN_COOKIE = "neko_token";
// Server-side (internal) API URL preferred; falls back to the public URL.
const API_BASE =
  process.env.NEKO_API_INTERNAL_URL ??
  process.env.NEXT_PUBLIC_API_BASE_URL ??
  "http://localhost:8080";

// Protect all app routes. Unauthenticated users go to /login. A present but
// stale/expired token (e.g. after a backend restart that invalidated it) is
// validated against the API and, if rejected, cleared so the user re-logs in
// instead of seeing demo data while mutations fail.
export async function middleware(req: NextRequest) {
  const token = req.cookies.get(TOKEN_COOKIE)?.value;
  const { pathname } = req.nextUrl;
  const isLogin = pathname === "/login";

  if (!token) {
    if (isLogin) return NextResponse.next();
    const url = req.nextUrl.clone();
    url.pathname = "/login";
    url.searchParams.set("next", pathname);
    return NextResponse.redirect(url);
  }

  // Validate the token (fail-open on network error so a transient API blip
  // doesn't lock everyone out).
  let valid = true;
  try {
    const res = await fetch(`${API_BASE}/api/v1/auth/me`, {
      headers: { Authorization: `Bearer ${token}` },
      signal: AbortSignal.timeout(4000),
      cache: "no-store",
    });
    if (res.status === 401) valid = false;
  } catch {
    valid = true; // fail open
  }

  if (!valid) {
    const url = req.nextUrl.clone();
    url.pathname = "/login";
    url.search = isLogin ? "" : `?next=${encodeURIComponent(pathname)}`;
    const resp = NextResponse.redirect(url);
    for (const c of ["neko_token", "neko_email", "neko_role"]) resp.cookies.delete(c);
    return resp;
  }

  if (isLogin) {
    const url = req.nextUrl.clone();
    url.pathname = "/";
    url.search = "";
    return NextResponse.redirect(url);
  }
  return NextResponse.next();
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
