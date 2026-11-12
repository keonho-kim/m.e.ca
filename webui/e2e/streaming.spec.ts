// Purpose: Verify streamed reasoning, saved Markdown and long-output scroll following.
// Pattern: Browser rendering contracts with controlled model output.
// Usage: MECA_UI_DESCRIPTOR points to the opt-in Go browser fixture.
// Related: webui/e2e/fixture.ts, cmd/meca/ui_provider_test.go.
import { expect } from '@playwright/test';
import {
  type BrowserFixture,
  browserFixture,
  connectBrowser,
  create,
  send,
  test,
} from '#e2e/fixture';

let fixture: BrowserFixture;
test.beforeAll(async () => {
  fixture = await browserFixture();
});
test.beforeEach(async ({ page }) => {
  await connectBrowser(page, fixture);
});
test('thinking streams as subdued Markdown and remains readable in saved history', async ({
  page,
}) => {
  await create(page);
  await send(page, 'THINKING_FIXTURE');
  const thinking = page.locator('.thinking');
  await expect(thinking.getByRole('heading', { name: 'Plan', exact: true })).toBeVisible();
  await expect(thinking.locator('strong')).toHaveText('Inspect');
  await expect(thinking).toContainText('the workspace');
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
  page,
}, info) => {
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await create(page);
  await send(page, 'LONG_STREAM_FIXTURE');
  await expect(page.locator('.history-rows')).toContainText('LONG_STREAM_FIXTURE');
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
