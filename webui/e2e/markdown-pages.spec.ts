// Purpose: Keep dense rich responses fully readable without an unbounded DOM or a full-text popover.
// Pattern: Browser contract over Astryx's public Markdown extension renderer.
// Usage: Run with MECA_UI_DESCRIPTOR pointing at TestUIBrowserFixture.
// Related: webui/src/shared/ui/markdown/pagination.tsx, webui/src/shared/ui/markdown/render-budget.ts.
import { readFile } from 'node:fs/promises';
import { expect, test } from '@playwright/test';

test('dense ordered responses expose their last entry inline with bounded rendered nodes', async ({
  page,
}) => {
  test.skip(!process.env.MECA_UI_DESCRIPTOR, 'start the isolated browser fixture first');
  const fixture = JSON.parse(await readFile(process.env.MECA_UI_DESCRIPTOR!, 'utf8'));
  const source = Array.from({ length: 600 }, (_, index) => `${index + 7}. Entry **${index}**`).join(
    '\n'
  );
  const errors: string[] = [];
  page.on('console', (message) => {
    if (message.text().includes('meca-render-pages')) errors.push(message.text());
  });
  await page.route('**/v1/command', async (route) => {
    if (route.request().postDataJSON().history)
      await route.fulfill({
        json: {
          history: { items: [{ sequence: '1', kind: 'message', role: 'assistant', text: source }] },
        },
      });
    else await route.continue();
  });
  await page.goto(`${fixture.url}/ui/`);
  await page
    .getByRole('textbox', { name: '서버 토큰', exact: true })
    .fill((await readFile(fixture.tokenFile, 'utf8')).trim());
  await page.getByRole('button', { name: '연결', exact: true }).click();
  await page
    .locator('.workspace-group')
    .filter({ has: page.locator('.workspace-name', { hasText: 'alpha' }) })
    .getByRole('button', { name: '새 대화', exact: true })
    .click();
  const controls = page.getByRole('group', { name: '본문 페이지', exact: true });
  await expect(controls).toBeVisible();
  const next = controls.getByRole('button', { name: '다음 페이지', exact: true });
  let pages = 0;
  while (await next.isEnabled()) {
    expect(await page.locator('.meca-prose *').count()).toBeLessThan(4000);
    expect(++pages).toBeLessThan(12);
    await next.click();
    await expect(controls).toBeVisible();
    expect(errors).toEqual([]);
  }
  await expect(page.getByText('599', { exact: true })).toBeVisible();
  expect(pages).toBeGreaterThan(0);
  expect(errors).toEqual([]);
  await expect(page.getByRole('dialog')).toHaveCount(0);
  await expect(page.getByRole('button', { name: /View complete text|원문 보기/ })).toHaveCount(0);
});
