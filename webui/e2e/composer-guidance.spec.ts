// Purpose: Keep send, steering and queue instructions readable as distinct platform-aware shortcut rows.
// Pattern: Browser interaction and accessible keyboard-hint contract.
// Usage: MECA_UI_DESCRIPTOR points to the isolated Go browser fixture.
// Related: webui/src/pages/workspace/ui/composer/send-hint.tsx, webui/e2e/fixture.ts.
import { expect } from '@playwright/test';
import { browserFixture, connectBrowser, create, input, test } from '#e2e/fixture';

test('the send tooltip separates send, steer and queue with native key badges', async ({
  page,
}) => {
  await connectBrowser(page, await browserFixture());
  await create(page);
  await input(page).fill('Shortcut guidance');
  await page.locator('.composer').getByRole('button', { name: '전송', exact: true }).hover();
  const tooltip = page.getByRole('tooltip');
  await expect(tooltip).toBeVisible();
  const rows = tooltip.locator('.composer-shortcut-row');
  await expect(rows).toHaveCount(3);
  await expect(rows.nth(0)).toContainText('전송');
  await expect(rows.nth(1)).toContainText('방향 수정');
  await expect(rows.nth(2)).toContainText('후속 예약');
  await expect(tooltip.locator('.astryx-kbd')).toHaveCount(3);
  const contrast = await tooltip
    .locator('kbd')
    .first()
    .evaluate((node) => {
      const tooltip = node.closest('[role="tooltip"]');
      if (!tooltip) throw Error('shortcut tooltip missing');
      const canvas = document.createElement('canvas');
      canvas.width = canvas.height = 1;
      const ctx = canvas.getContext('2d');
      if (!ctx) throw Error('color measurement unavailable');
      const luminance = (pixels: Uint8ClampedArray) =>
        [...pixels]
          .slice(0, 3)
          .map((v) => {
            const c = v / 255;
            return c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
          })
          .reduce((sum, value, index) => sum + value * ([0.2126, 0.7152, 0.0722][index] ?? 0), 0);
      ctx.fillStyle = getComputedStyle(tooltip).backgroundColor;
      ctx.fillRect(0, 0, 1, 1);
      ctx.fillStyle = getComputedStyle(node).backgroundColor;
      ctx.fillRect(0, 0, 1, 1);
      const background = luminance(ctx.getImageData(0, 0, 1, 1).data);
      ctx.clearRect(0, 0, 1, 1);
      ctx.fillStyle = getComputedStyle(node).color;
      ctx.fillRect(0, 0, 1, 1);
      const foreground = luminance(ctx.getImageData(0, 0, 1, 1).data);
      return (Math.max(foreground, background) + 0.05) / (Math.min(foreground, background) + 0.05);
    });
  expect(contrast).toBeGreaterThanOrEqual(4.5);
  const boxes = await rows.evaluateAll((items) =>
    items.map((item) => {
      const box = item.getBoundingClientRect();
      return { top: box.top, bottom: box.bottom };
    })
  );
  expect(boxes[1]?.top).toBeGreaterThanOrEqual(boxes[0]?.bottom ?? Infinity);
  expect(boxes[2]?.top).toBeGreaterThanOrEqual(boxes[1]?.bottom ?? Infinity);
  await page.keyboard.press('Escape');
  await expect(tooltip).toBeHidden();
  await expect(input(page)).toHaveText('Shortcut guidance');
});
