// Purpose: Run browser performance measurement and live provider checks on demand.
// Pattern: Composite Playwright configuration sharing isolated server setup.
// Usage: bun run test:performance or bun run test:live-provider.
// Related: webui/playwright.config.ts, webui/e2e/web-vitals.spec.ts, webui/e2e/live-provider.spec.ts.
import { defineConfig } from '@playwright/test';
import browserTests from '#e2e/config';

export default defineConfig({
  ...browserTests,
  testIgnore: [],
  reporter: 'line',
  projects: [
    {
      name: 'performance',
      testMatch: '**/web-vitals.spec.ts',
      use: { browserName: 'chromium', channel: 'chromium' },
    },
    {
      name: 'live-provider',
      testMatch: '**/live-provider.spec.ts',
      use: { browserName: 'chromium', channel: 'chromium' },
    },
  ],
});
