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
  use: {
    channel: 'chrome',
    locale: 'ko-KR',
    viewport: { width: 1440, height: 1000 },
    trace: 'off',
    screenshot: 'only-on-failure',
  },
});
