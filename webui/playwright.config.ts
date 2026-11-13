// Purpose: Run isolated embedded-browser contracts against the opt-in Go server fixture.
// Pattern: Browser test configuration.
// Usage: MECA_UI_DESCRIPTOR=/tmp/task/connection.json bun run test:e2e.
// Related: cmd/meca/ui_browser_test.go, webui/e2e/workspace.spec.ts.
import { defineConfig } from '@playwright/test';
export default defineConfig({
  testDir: 'e2e',
  workers: 1,
  fullyParallel: false,
  timeout: 60000,
  reporter: 'line',
  projects: [
    { name: 'chromium', use: { browserName: 'chromium', channel: 'chrome' } },
    {
      name: 'firefox',
      use: { browserName: 'firefox' },
      testMatch: /browser-(contracts|quality|features|policy)\.spec\.ts/,
    },
    {
      name: 'webkit',
      use: { browserName: 'webkit' },
      testMatch: /browser-(contracts|quality|features|policy)\.spec\.ts/,
    },
  ],
  use: {
    ignoreHTTPSErrors: process.env.MECA_UI_TLS === '1',
    locale: 'ko-KR',
    viewport: { width: 1440, height: 1000 },
    trace: 'off',
    screenshot: 'only-on-failure',
  },
});
