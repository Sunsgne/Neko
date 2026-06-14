import { cookies } from "next/headers";
import { TOKEN_COOKIE, EMAIL_COOKIE, ROLE_COOKIE } from "@/lib/cookies";

// Server-only helpers to read the session from cookies during SSR.

export function serverToken(): string | undefined {
  return cookies().get(TOKEN_COOKIE)?.value;
}

export function serverIdentity(): { email?: string; role?: string } {
  const jar = cookies();
  return {
    email: jar.get(EMAIL_COOKIE)?.value,
    role: jar.get(ROLE_COOKIE)?.value,
  };
}
