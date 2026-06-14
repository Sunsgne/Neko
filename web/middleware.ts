import { NextRequest, NextResponse } from "next/server";

const TOKEN_COOKIE = "neko_token";

// Protect all app routes: unauthenticated users are redirected to /login.
// The /login route is public; authenticated users visiting it go to the
// dashboard.
export function middleware(req: NextRequest) {
  const token = req.cookies.get(TOKEN_COOKIE)?.value;
  const { pathname } = req.nextUrl;
  const isLogin = pathname === "/login";

  if (!token && !isLogin) {
    const url = req.nextUrl.clone();
    url.pathname = "/login";
    url.searchParams.set("next", pathname);
    return NextResponse.redirect(url);
  }
  if (token && isLogin) {
    const url = req.nextUrl.clone();
    url.pathname = "/";
    url.search = "";
    return NextResponse.redirect(url);
  }
  return NextResponse.next();
}

export const config = {
  // Run on all paths except Next internals and static assets.
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
