// Purpose: Verify streamed reasoning, saved Markdown and long-output scroll following.
// Pattern: Browser rendering contracts with controlled model output.
// Usage: Run through the Playwright browser suite.
// Related: webui/e2e/fixture.ts, cmd/meca/ui_provider_test.go.
import { expect } from '@playwright/test';
import { connectBrowser, create, rpc, send, test } from '#e2e/fixture';

test.beforeEach(async ({ page, server: fixture }) => {
  await connectBrowser(page, fixture);
});
test('thinking streams as subdued Markdown and remains readable in saved history', async ({
  server: fixture,
  page,
}) => {
  await create(page);
  await send(page, 'THINKING_FIXTURE');
  const thinking = page.locator('.thinking');
  await expect(thinking.getByRole('heading', { name: 'Plan', exact: true })).toBeVisible();
  await expect(thinking.locator('strong')).toHaveText('Inspect');
  await expect(thinking).toContainText('the workspace');
  await expect(page.locator('.live-output .astryx-status-dot')).toHaveCount(0);
  await expect(page.getByRole('log')).toHaveAttribute('aria-busy', 'true');
  await expect(page.getByRole('button', { name: '작업 중지', exact: true })).toBeVisible();
  await expect(page.locator('.history-rows')).not.toContainText('Completed: THINKING_FIXTURE');
  for (const theme of ['light', 'dark']) {
    await expect(page.locator('html')).toHaveAttribute('data-theme', theme);
    expect(
      await thinking.locator('[data-tone=thinking]').evaluate((el) => {
        const style = getComputedStyle(el),
          reference = document.createElement('span');
        reference.style.color = 'var(--color-text-secondary)';
        el.append(reference);
        const expected = getComputedStyle(reference).color;
        reference.remove();
        return style.color === expected;
      })
    ).toBe(true);
    if (theme === 'light')
      await page.getByRole('button', { name: '화면 테마', exact: true }).click();
  }
  await page.request.post(`${fixture.provider}/test/release-thinking`);
  await expect(page.locator('.history-rows')).toContainText('Completed: THINKING_FIXTURE');
  await expect(page.locator('.history-rows .thinking')).toContainText('and verify the result.');
  await page.reload();
  await expect(page.locator('.history-rows .thinking')).toContainText('and verify the result.');
});

test('long live Markdown follows the newest output after durable history has settled', async ({
  server: fixture,
  page,
}, info) => {
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await create(page);
  await send(page, 'LONG_STREAM_FIXTURE');
  const path = new URL(page.url()).pathname.split('/');
  await expect
    .poll(async () =>
      (
        await rpc(page, { history: { sessionId: path[5], limit: 32 } }, path[3])
      ).history?.items?.some((item) => item.message?.text === 'LONG_STREAM_FIXTURE')
    )
    .toBe(true);
  await expect(page.getByRole('log')).toContainText('LONG_STREAM_FIXTURE');
  await expect(page.locator('.live-output')).toContainText('Fixture');
  const scroller = page.locator('.workspace-body');
  await scroller.evaluate((el) => {
    el.scrollTop = el.scrollHeight;
  });
  const gap = () => scroller.evaluate((el) => el.scrollHeight - el.clientHeight - el.scrollTop);
  await expect.poll(gap).toBeLessThan(30);
  await page.request.post(`${fixture.provider}/test/expand-stream`);
  await expect.poll(() => scroller.evaluate((el) => el.scrollHeight)).toBeGreaterThan(3000);
  await expect.poll(gap).toBeLessThan(30);
  await page.screenshot({
    path: info.outputPath('streaming-long.png'),
    fullPage: true,
    animations: 'disabled',
  });
  await page.getByRole('button', { name: '작업 중지', exact: true }).click();
  await expect(page.getByRole('button', { name: '작업 중지', exact: true })).toBeHidden();
});
