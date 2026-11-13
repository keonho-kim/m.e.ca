// Purpose: Verify the authenticated loading surface while preserving deferred workspace downloads.
// Pattern: Controlled module-read gate over the real embedded application.
// Usage: MECA_UI_DESCRIPTOR points to the isolated Go browser fixture.
// Related: webui/src/app/routes/application.tsx, webui/src/pages/connect/ui/connection-loading.tsx.
import { readFile } from 'node:fs/promises';
import { expect } from '@playwright/test';
import { browserFixture, connectBrowser, test } from '#e2e/fixture';

for (const dark of [false, true])
  test(`workspace loading uses a readable glass surface in ${dark ? 'dark' : 'light'} mode`, async ({
    page,
  }) => {
    const fixture = await browserFixture();
    if (dark) {
      await connectBrowser(page, fixture);
      await page.getByRole('button', { name: '화면 테마', exact: true }).click();
    }
    let release = () => {};
    const held = new Promise<void>((resolve) => {
      release = resolve;
    });
    await page.route('**/ui/assets/workspace-*.js', async (route) => {
      await held;
      await route.continue();
    });
    const entering = page.waitForRequest('**/ui/assets/workspace-*.js');
    try {
      if (dark) await page.reload();
      else {
        await page.goto(`${fixture.url}/ui/`);
        const token = (await readFile(fixture.tokenFile, 'utf8')).trim();
        await page.getByRole('textbox', { name: /^서버 토큰/ }).fill(token);
        await page.getByRole('button', { name: '연결', exact: true }).click();
      }
      await entering;
      const loading = page.locator('.startup-screen');
      await expect(loading).toHaveAttribute('role', 'status');
      await expect(loading).toContainText('작업 공간을 준비하고 있습니다.');
      await page.setViewportSize({ width: 390, height: 844 });
      const style = await loading.locator('.startup-surface').evaluate((node) => {
        const style = getComputedStyle(node),
          box = node.getBoundingClientRect();
        return {
          blur: style.backdropFilter,
          left: box.left,
          right: box.right,
          reduced: matchMedia('(prefers-reduced-transparency: reduce)').matches,
          background: style.backgroundColor,
          supports: CSS.supports('backdrop-filter', style.getPropertyValue('--meca-glass-filter')),
        };
      });
      if (style.reduced || !style.supports) {
        expect(style.blur).toBe('none');
        expect(style.background).toMatch(/^rgb\(/);
      } else expect(style.blur).toContain('blur');
      expect(style.left).toBeGreaterThanOrEqual(0);
      expect(style.right).toBeLessThanOrEqual(390);
      release();
      await expect(page.getByRole('tab', { name: '대화', exact: true })).toBeVisible();
      await expect(loading).toBeHidden();
    } finally {
      release();
    }
  });
