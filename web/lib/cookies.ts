// Plain (non-client, non-server) cookie name constants shared by both the
// client session helpers and server-side readers. Keeping these out of a
// "use client" module is essential: importing a "use client" export into a
// Server Component turns it into a client reference (not the literal value),
// which previously broke server-side cookie reads.
export const TOKEN_COOKIE = "neko_token";
export const EMAIL_COOKIE = "neko_email";
export const ROLE_COOKIE = "neko_role";
