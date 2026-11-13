// Purpose: Enforce accessible navigation and bounded presentation across supported browser engines.
// Pattern: Isolated browser audit with development-only axe and local Web Vitals measurement.
// Usage: MECA_UI_DESCRIPTOR points to TestUIBrowserFixture; no metrics leave the browser.
// Related: webui/playwright.config.ts, webui/src/app/styles/theme.ts.

import { readFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import AxeBuilder from '@axe-core/playwright';
import { expect } from '@playwright/test';
import type { Metric } from 'web-vitals';
import { browserFixture, connectBrowser, create, input, test } from '#e2e/fixture';

test('workspace and model form expose keyboard, heading and validation semantics', async ({
  page,
}) => {
  await connectBrowser(page, await browserFixture());
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
  page,
}) => {
  await connectBrowser(page, await browserFixture());
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

test('chat reports local interaction and layout metrics without network collection', async ({
  page,
  browserName,
}, testInfo) => {
  test.skip(browserName !== 'chromium', 'INP instrumentation uses Chromium Event Timing');
  const runtime = await readFile(
    join(dirname(fileURLToPath(import.meta.resolve('web-vitals'))), 'web-vitals.iife.js'),
    'utf8'
  );
  await page.addInitScript({
    content: `${runtime}\nwindow.__mecaVitals = {}; for (const name of ['CLS','INP','LCP']) webVitals['on'+name](value => { window.__mecaVitals[name] = {name: value.name, value: value.value}; }, {reportAllChanges:true});`,
  });
  await connectBrowser(page, await browserFixture());
  await create(page);
  await input(page).fill('Input responsiveness fixture');
  await page.getByRole('tab', { name: '파일', exact: true }).click();
  await page.getByRole('button', { name: 'sample.go', exact: true }).click();
  await page.getByRole('tab', { name: '대화', exact: true }).click();
  await expect(input(page)).toHaveText('Input responsiveness fixture');
  const metrics = await page.evaluate(
    () =>
      (window as unknown as { __mecaVitals: Record<string, Pick<Metric, 'name' | 'value'>> })
        .__mecaVitals
  );
  await testInfo.attach('local-web-vitals', {
    body: JSON.stringify(metrics),
    contentType: 'application/json',
  });
  expect(metrics.CLS?.value ?? 0).toBeLessThan(0.1);
  if (metrics.INP) expect(metrics.INP.value).toBeLessThan(250);
});
