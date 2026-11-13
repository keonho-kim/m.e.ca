// Purpose: Exercise real Chromium back/forward-cache restoration with the browser feature enabled.
// Pattern: Native navigation regression test with observed persisted pageshow.
// Usage: The Chromium project runs this against TestUIBrowserFixture.
// Related: webui/src/pages/workspace/api/observer.ts, webui/e2e/reconciliation.spec.ts.
import { expect } from '@playwright/test';
import { browserFixture, connectBrowser, create, rpc, test } from '#e2e/fixture';

// Playwright disables this browser feature by default; synthetic events alone
// cannot establish that the real retained page restores its connection.
test.use({ launchOptions: { ignoreDefaultArgs: ['--disable-back-forward-cache'] } });

test('a cached page reauthenticates and reads a fresh snapshot without resubmitting work', async ({
  page,
}) => {
  const fixture = await browserFixture();
  await page.addInitScript(() => {
    window.addEventListener('pageshow', (event) => {
      Object.assign(window, { restoredFromCache: event.persisted });
    });
  });
  await connectBrowser(page, fixture);
  await create(page);
  const current = page.url();
  const workspace = new URL(current).pathname.split('/')[3] ?? '';
  await expect(page.locator('.execution-status')).toContainText(/입력 0.*출력 0/);
  await page.goto(`${fixture.url}/ui/favicon.svg`);
  let probes = 0;
  page.on('request', (request) => {
    if (new URL(request.url()).pathname === '/v1/ui/session') probes++;
  });
  // A retained page does not emit a second load event.
  await page.goBack({ waitUntil: 'commit' });
  await page.waitForURL(current, { waitUntil: 'commit' });
  await expect.poll(() => page.evaluate(() => Reflect.get(window, 'restoredFromCache'))).toBe(true);
  await expect.poll(() => probes).toBeGreaterThan(0);
  await expect(page.getByRole('tab', { name: '대화', exact: true })).toBeVisible();
  const response = await rpc(
    page,
    { state: { sessionId: new URL(current).pathname.split('/').at(-1) } },
    workspace
  );
  expect(response.state?.tasks?.length ?? 0).toBe(0);
});
