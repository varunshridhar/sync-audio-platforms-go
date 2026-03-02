/** @type {import('next').NextConfig} */
// Next.js runtime config:
// - hide "X-Powered-By" header for security hygiene,
// - enable typed routes for safer navigation in TypeScript.
const nextConfig = {
  poweredByHeader: false,
  experimental: {
    typedRoutes: true
  }
};

export default nextConfig;

