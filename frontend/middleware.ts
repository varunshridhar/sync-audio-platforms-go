import { NextRequest, NextResponse } from "next/server";

// Frontend middleware adds response security headers for all app routes.
export function middleware(_request: NextRequest) {
  const response = NextResponse.next();
  response.headers.set("X-Content-Type-Options", "nosniff");
  response.headers.set("X-Frame-Options", "DENY");
  response.headers.set("Referrer-Policy", "strict-origin-when-cross-origin");
  response.headers.set("Permissions-Policy", "geolocation=(), microphone=(), camera=()");
  response.headers.set(
    "Content-Security-Policy",
    "default-src 'self'; script-src 'self' https://challenges.cloudflare.com; frame-src 'self' https://challenges.cloudflare.com; frame-ancestors 'none'; base-uri 'self';"
  );
  return response;
}

export const config = {
  // Exclude static assets from middleware for performance.
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"]
};

