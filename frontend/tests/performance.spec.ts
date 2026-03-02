import { expect, test } from "@playwright/test";

type PerfRoute = {
  path: string;
  maxDomContentLoadedMs: number;
  maxLoadEventMs: number;
};

const routes: PerfRoute[] = [
  { path: "/", maxDomContentLoadedMs: 1500, maxLoadEventMs: 2500 },
  { path: "/login", maxDomContentLoadedMs: 1800, maxLoadEventMs: 3000 },
  { path: "/dashboard", maxDomContentLoadedMs: 1800, maxLoadEventMs: 3000 }
];

test.describe("page load performance", () => {
  for (const route of routes) {
    test(`measures ${route.path} load timing`, async ({ page }) => {
      await page.goto(route.path, { waitUntil: "load" });

      const metrics = await page.evaluate(() => {
        const nav = performance.getEntriesByType("navigation")[0] as PerformanceNavigationTiming;
        return {
          domContentLoadedMs: nav.domContentLoadedEventEnd,
          loadEventMs: nav.loadEventEnd
        };
      });

      console.log(
        `${route.path} -> DCL ${Math.round(metrics.domContentLoadedMs)}ms | LOAD ${Math.round(metrics.loadEventMs)}ms`
      );

      expect(metrics.domContentLoadedMs).toBeLessThan(route.maxDomContentLoadedMs);
      expect(metrics.loadEventMs).toBeLessThan(route.maxLoadEventMs);
    });
  }
});
