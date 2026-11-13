// Purpose: Guard observer capacity, cached-module recovery and virtualized keyboard ownership.
// Pattern: Browser regressions against the embedded isolated fixture.
// Usage: Runs on Chromium, Firefox and WebKit with no user projects or model credentials.
// Related: internal/serverapi/ui_observer.go, webui/src/shared/ui/recovery/index.tsx.
import { expect } from '@playwright/test';
import { browserFixture, connectBrowser, create, historyPage, test } from '#e2e/fixture';

test('three file tabs leave HTTP requests available', async ({ page, context }) => {
  const fixture = await browserFixture();
  await connectBrowser(page, fixture);
  const peers = [page];
  try {
    for (let index = 0; index < 3; index++) {
      const peer = index === 0 ? page : await context.newPage();
      if (index > 0) {
        peers.push(peer);
        await peer.goto(`${fixture.url}/ui/`);
      }
      await peer.getByRole('tab', { name: '파일', exact: true }).click();
      await expect(peer.getByRole('button', { name: 'sample.go', exact: true })).toBeVisible();
    }
    const result = await page.evaluate(async () => {
      const response = await fetch('/v1/ui/session', { signal: AbortSignal.timeout(3000) });
      const body = await response.json();
      const request = performance
        .getEntriesByName(new URL('/v1/ui/session', location.href).href)
        .at(-1) as PerformanceResourceTiming | undefined;
      return { ok: response.ok, body, protocol: request?.nextHopProtocol };
    });
    expect(result.ok).toBe(true);
    expect(result.body.authenticated).toBe(true);
    if (fixture.url.startsWith('https:')) expect(result.protocol).toBe('h2');
  } finally {
    for (const peer of peers.slice(1)) await peer.close();
  }
});

test('a missing lazy module can recover with an independent reload action', async ({ page }) => {
  const fixture = await browserFixture();
  await page.route('**/ui/assets/workspace-*.js', (route) =>
    route.fulfill({ status: 404, body: 'missing module' })
  );
  // Authenticate without waiting for the deliberately unavailable workspace.
  const { readFile } = await import('node:fs/promises');
  const response = await page.request.post(`${fixture.url}/v1/ui/login`, {
    headers: {
      'X-Meca-UI': '1',
      Origin: fixture.url,
      Authorization: `Bearer ${(await readFile(fixture.tokenFile, 'utf8')).trim()}`,
    },
  });
  expect(response.ok()).toBe(true);
  await page.goto(`${fixture.url}/ui/`);
  const reload = page.getByRole('button', { name: /화면 다시 불러오기|Reload application/ });
  await expect(reload).toBeVisible();
  await page.unroute('**/ui/assets/workspace-*.js');
  await reload.click();
  await expect(page.getByRole('tab', { name: /대화|Chat/, exact: true })).toBeVisible();
});

test('focused tool rows stay mounted while distant history is visible', async ({ page }) => {
  await page.route('**/v1/command', (route) => {
    const cmd = route.request().postDataJSON();
    if (cmd.history)
      return route.fulfill({
        json: {
          history: historyPage(
            Array.from({ length: 64 }, (_, index) => ({
              sequence: String(index + 1),
              event: {
                kind: 'tool_succeeded',
                tool: {
                  name: `focus_tool_${index}`,
                  phase: 'TOOL_PHASE_SUCCEEDED',
                  summary: 'fixture',
                },
              },
            }))
          ),
        },
      });
    return route.continue();
  });
  await connectBrowser(page, await browserFixture());
  await create(page);
  const trigger = page.locator('.tool-disclosure > button').first();
  const name = await trigger.evaluate((node) => {
    node.focus();
    return node.textContent;
  });
  await trigger.evaluate((node) => {
    let parent = node.parentElement;
    while (
      parent &&
      !(
        parent.scrollHeight > parent.clientHeight &&
        /auto|scroll/.test(getComputedStyle(parent).overflowY)
      )
    )
      parent = parent.parentElement;
    if (!parent) throw Error('Missing scroll owner');
    parent.scrollTop = parent.scrollHeight;
  });
  await expect.poll(() => page.evaluate(() => document.activeElement?.textContent)).toBe(name);
  await expect(page.locator('.tool-entry').filter({ hasText: name })).toHaveCount(1);
});
