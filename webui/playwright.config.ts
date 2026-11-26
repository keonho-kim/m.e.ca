// Purpose: Run browser behavior tests against isolated embedded MECA servers.
// Pattern: Browser test configuration.
// Usage: bun run test:e2e --project=chromium after building the embedded UI.
// Related: webui/e2e/prepare.ts, webui/e2e/required.ts, cmd/meca/ui_browser_test.go.
import { defineConfig } from '@playwright/test';

const portable =
  /(browser-(contracts|quality|features|policy)|new-chat|chat-stream-order|team-(filter|presentation))\.spec\.ts/;

export default defineConfig({
  testDir: 'e2e',
  testIgnore: ['**/web-vitals.spec.ts', '**/live-provider.spec.ts'],
  globalSetup: './e2e/prepare.ts',
  workers: 1,
  timeout: 60000,
  reporter: [['line'], ['./e2e/required.ts']],
  projects: [
    { name: 'chromium', use: { browserName: 'chromium', channel: 'chromium' } },
    { name: 'firefox', use: { browserName: 'firefox' }, testMatch: portable },
    { name: 'webkit', use: { browserName: 'webkit' }, testMatch: portable },
  ],
  use: {
    ignoreHTTPSErrors: process.env.MECA_UI_TLS === '1',
    locale: 'ko-KR',
    viewport: { width: 1440, height: 1000 },
    trace: 'off',
    screenshot: 'only-on-failure',
  },
});
