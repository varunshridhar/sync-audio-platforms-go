import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./tests",
  timeout: 30_000,
  retries: 0,
  use: {
    baseURL: "http://127.0.0.1:3300",
    browserName: "chromium",
    headless: true,
    trace: "off"
  },
  webServer: {
    command: "npm run dev -- --hostname 127.0.0.1 --port 3300",
    url: "http://127.0.0.1:3300",
    reuseExistingServer: true,
    timeout: 120_000
  }
});
