// Purpose: Verify breadcrumb depth/overflow, folder chevron alignment and language-aware file icons.
// Pattern: Rendered directory navigation contracts with real filesystem fixtures.
// Usage: MECA_UI_DESCRIPTOR points to the isolated Go browser fixture.
// Related: webui/src/pages/workspace/ui/files/directory-breadcrumbs.tsx, webui/src/shared/ui/file-icon/index.tsx.
import { mkdir, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { expect } from '@playwright/test';
import { type BrowserFixture, browserFixture, connectBrowser, test } from '#e2e/fixture';

let fixture: BrowserFixture;
test.beforeAll(async () => {
  fixture = await browserFixture();
  await mkdir(join(fixture.projectA, '00-breadcrumb/one/two/three/four/five'), { recursive: true });
  for (const name of ['a.component.ts', 'App.vue', 'App.tsx', 'Dockerfile', 'file.constructor'])
    await writeFile(join(fixture.projectA, '00-breadcrumb', name), '// file icon fixture');
});
test('directory trails collapse leading depth within their panel and keep arrows centered', async ({
  page,
}) => {
  await connectBrowser(page, fixture);
  await page.getByRole('tab', { name: '파일', exact: true }).click();
  await page.getByRole('button', { name: '00-breadcrumb', exact: true }).click();
  const folder = page.getByRole('button', { name: 'one', exact: true });
  const gap = await folder.evaluate((node) => {
    const arrow = [...node.querySelectorAll('svg')].at(-1);
    const walker = document.createTreeWalker(node, NodeFilter.SHOW_TEXT);
    for (let text = walker.nextNode(); text; text = walker.nextNode()) {
      if (text.textContent?.trim() !== 'one' || !arrow) continue;
      const range = document.createRange();
      range.selectNodeContents(text);
      const label = range.getBoundingClientRect(),
        icon = arrow.getBoundingClientRect();
      return Math.abs(label.y + label.height / 2 - icon.y - icon.height / 2);
    }
    throw Error('folder name or arrow missing');
  });
  expect(gap).toBeLessThanOrEqual(2);
  for (const type of ['angular', 'vue', 'react', 'docker'])
    await expect(page.locator(`[data-file-type="${type}"]`)).toHaveCount(1);
  await expect(page.getByRole('button', { name: 'file.constructor', exact: true })).toBeVisible();
  for (const name of ['one', 'two', 'three', 'four', 'five'])
    await page.getByRole('button', { name, exact: true }).click();
  const trail = page.locator('.directory-breadcrumbs');
  await expect(trail).toContainText('five');
  await expect(trail).toContainText('…');
  expect(await trail.locator('ol > li').count()).toBeLessThanOrEqual(4);
  expect(await trail.evaluate((node) => node.scrollWidth <= node.clientWidth + 1)).toBe(true);
  const ratio = await page.locator('.directory-location').evaluate((node) => {
    const trail = node.querySelector('.directory-breadcrumbs');
    if (!trail) throw Error('directory breadcrumb missing');
    return trail.getBoundingClientRect().width / node.getBoundingClientRect().width;
  });
  expect(ratio).toBeCloseTo(0.8, 1);
  await page.setViewportSize({ width: 1100, height: 850 });
  expect(
    await page
      .locator('.directory-column')
      .evaluate((node) => node.scrollWidth <= node.clientWidth + 1)
  ).toBe(true);
  await trail.getByRole('button').first().click();
  await page.getByRole('menuitem', { name: 'one', exact: true }).click();
  await expect(page.getByRole('button', { name: 'two', exact: true })).toBeVisible();
});
