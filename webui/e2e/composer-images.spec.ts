// Purpose: Verify image islands, native hover removal and preview URL cleanup across drafts.
// Pattern: Browser-to-engine composer resource contracts.
// Usage: MECA_UI_DESCRIPTOR points to the isolated embedded UI fixture.
// Related: webui/src/pages/workspace/ui/composer/composer.tsx, cmd/meca/ui_browser_test.go.

import { expect, type Page } from '@playwright/test';
import { type BrowserFixture, browserFixture, connectBrowser, required, test } from '#e2e/fixture';

let fixture: BrowserFixture;
const png = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR4nGMQMgn7DwAClAGcHVtGXwAAAABJRU5ErkJggg==',
  'base64'
);
async function create(page: Page) {
  const previous = page.url();
  await page
    .locator('.workspace-group')
    .filter({ has: page.locator('.workspace-name', { hasText: 'alpha' }) })
    .getByRole('button', { name: '새 대화', exact: true })
    .click();
  await page.waitForURL((url) => url.href !== previous && url.pathname.includes('/c/session_'));
  await expect(page.getByRole('combobox', { name: '작업 입력', exact: true })).toBeEnabled();
}
test.beforeAll(async () => {
  fixture = await browserFixture();
});
test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => {
    const create = URL.createObjectURL.bind(URL),
      revoke = URL.revokeObjectURL.bind(URL);
    const live = new Set<string>();
    let created = 0;
    Object.assign(window, { previewUsage: () => ({ live: live.size, created }) });
    URL.createObjectURL = (blob) => {
      const url = create(blob);
      live.add(url);
      created++;
      return url;
    };
    URL.revokeObjectURL = (url) => {
      live.delete(url);
      revoke(url);
    };
  });
  await connectBrowser(page, fixture);
  await create(page);
});
const usage = (page: Page) =>
  page.evaluate(() =>
    (window as unknown as { previewUsage: () => { live: number; created: number } }).previewUsage()
  );
async function attach(page: Page, count: number) {
  await page.locator('input[type=file]').setInputFiles(
    Array.from({ length: count }, (_, i) => ({
      name: `screen-${i + 1}.png`,
      mimeType: 'image/png',
      buffer: png,
    }))
  );
}
test('images form one native carousel above the composer with hover and keyboard removal', async ({
  page,
}) => {
  await attach(page, 8);
  const carousel = page.getByRole('region', { name: '첨부 이미지', exact: true });
  await expect(carousel).toBeVisible();
  await expect(carousel.locator('.astryx-thumbnail')).toHaveCount(8);
  const thumbs = await carousel.locator('.astryx-thumbnail').all();
  const first = (await thumbs[0]?.boundingBox().then(required))!;
  for (const thumb of thumbs) {
    const rect = await thumb.boundingBox().then(required);
    expect(rect.width).toBeLessThanOrEqual(80);
    expect(Math.abs(rect.y - first.y)).toBeLessThan(2);
  }
  const frame = await carousel.boundingBox().then(required),
    composer = await page.locator('.composer').boundingBox().then(required);
  expect(frame.y + frame.height).toBeLessThanOrEqual(composer.y);
  const cards = await carousel.locator('.composer-image').all();
  expect(cards).toHaveLength(8);
  for (const direction of ['ltr', 'rtl']) {
    await carousel.evaluate((element, dir) => element.setAttribute('dir', dir), direction);
    for (const card of cards) {
      const thumbnail = await card.locator('.astryx-thumbnail').boundingBox().then(required);
      const action = await card.getByRole('button').boundingBox().then(required);
      const topInset = action.y - thumbnail.y;
      const rightInset = thumbnail.x + thumbnail.width - action.x - action.width;
      expect(topInset).toBeGreaterThanOrEqual(0);
      expect(topInset).toBeLessThan(action.height / 2);
      expect(rightInset).toBeGreaterThanOrEqual(0);
      expect(rightInset).toBeLessThan(action.width / 2);
    }
  }
  await carousel.evaluate((element) => element.removeAttribute('dir'));
  const image = carousel.getByRole('group', { name: /^screen-1\.png/ });
  await image.hover();
  const remove = carousel.getByRole('button', { name: /screen-1\.png/ });
  await expect(remove).toBeVisible();
  const circle = await remove.evaluate((element) => {
    const rect = element.getBoundingClientRect();
    return {
      width: rect.width,
      height: rect.height,
      radius: parseFloat(getComputedStyle(element).borderTopLeftRadius),
    };
  });
  expect(circle.width).toBe(circle.height);
  expect(circle.radius).toBeGreaterThanOrEqual(circle.width / 2);
  await remove.click();
  await expect(carousel.locator('.astryx-thumbnail')).toHaveCount(7);
  const next = carousel.getByRole('button', { name: /screen-2\.png/ });
  await next.focus();
  await expect(next).toBeVisible();
  await next.press('Enter');
  await expect(carousel.locator('.astryx-thumbnail')).toHaveCount(6);
  await page.setViewportSize({ width: 390, height: 844 });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
});
test('typing reuses previews and switching drafts releases inactive preview URLs', async ({
  page,
}) => {
  const conversation = page.url();
  await attach(page, 2);
  await expect.poll(() => usage(page)).toEqual({ live: 2, created: 2 });
  await page
    .getByRole('combobox', { name: '작업 입력', exact: true })
    .fill('Keep the image previews stable');
  expect(await usage(page)).toEqual({ live: 2, created: 2 });
  await create(page);
  await expect.poll(async () => (await usage(page)).live).toBe(0);
  await page.goBack();
  await page.waitForURL(conversation);
  await expect(
    page.getByRole('region', { name: '첨부 이미지', exact: true }).locator('.astryx-thumbnail')
  ).toHaveCount(2);
  await expect.poll(async () => (await usage(page)).live).toBe(2);
});
