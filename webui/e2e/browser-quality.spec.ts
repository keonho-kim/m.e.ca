// Purpose: Enforce accessible navigation and bounded presentation across supported browser engines.
// Pattern: Browser accessibility and responsive layout checks.
// Usage: Run through the Playwright browser suite.
// Related: webui/playwright.config.ts, webui/src/app/styles/theme.ts.

import AxeBuilder from '@axe-core/playwright';
import { expect } from '@playwright/test';
import { connectBrowser, create, input, test } from '#e2e/fixture';

test('workspace and model form expose keyboard, heading and validation semantics', async ({
  server: fixture,
  page,
}) => {
  await connectBrowser(page, fixture);
  await create(page);
  await expect(page.getByRole('main')).toHaveCount(1);
  await expect(page.getByRole('heading', { level: 1 })).toHaveCount(1);
  await expect(page.getByRole('link', { name: /대화 열기/ }).first()).toHaveAttribute(
    'href',
    /\/ui\/w\/.+\/c\//
  );
  await page.getByRole('button', { name: '설정', exact: true }).click();
  const endpoint = page.getByRole('textbox', { name: /Base URL/ });
  await endpoint.fill('https://');
  await page.getByRole('button', { name: '연결 진단', exact: true }).click();
  await expect(endpoint).toBeFocused();
  await expect(endpoint).toHaveAttribute('aria-invalid', 'true');
  await expect(
    page.getByText('올바른 HTTP 또는 HTTPS 서버 주소를 입력해 주세요.', { exact: true })
  ).toBeVisible();
  const result = await new AxeBuilder({ page })
    .withTags(['wcag2a', 'wcag2aa', 'wcag21aa', 'wcag22aa'])
    .analyze();
  expect(
    result.violations.map((item) => ({
      id: item.id,
      targets: item.nodes.map((node) => node.target),
    }))
  ).toEqual([]);
});

test('narrow and reduced-motion views retain all controls without horizontal overflow', async ({
  server: fixture,
  page,
}) => {
  await connectBrowser(page, fixture);
  await create(page);
  await page.emulateMedia({ reducedMotion: 'reduce' });
  for (const width of [1440, 768, 390]) {
    await page.setViewportSize({ width, height: 900 });
    await expect(input(page)).toBeVisible();
    const geometry = await page.evaluate(() => ({
      width: innerWidth,
      scroll: document.documentElement.scrollWidth,
      nodes: document.querySelectorAll('*').length,
    }));
    expect(geometry.scroll).toBeLessThanOrEqual(geometry.width);
    expect(geometry.nodes).toBeLessThan(1800);
  }
});
