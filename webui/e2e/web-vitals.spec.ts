// Purpose: Measure observed interaction and layout performance in isolated Chromium.
// Pattern: On-demand browser performance measurement.
// Usage: bun run test:performance after building the embedded UI.
// Related: webui/playwright.probes.config.ts, webui/e2e/browser-quality.spec.ts.
import { readFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { expect } from '@playwright/test';
import type { Metric } from 'web-vitals';
import { connectBrowser, create, input, test } from '#e2e/fixture';

test('editing and switching panels produces local Web Vitals measurements', async ({
  page,
  server,
}, info) => {
  const runtime = await readFile(
    join(dirname(fileURLToPath(import.meta.resolve('web-vitals'))), 'web-vitals.iife.js'),
    'utf8'
  );
  await page.addInitScript({
    content: `${runtime}
      window.__mecaVitals = {};
      const record = ({name, value}) => { window.__mecaVitals[name] = value; };
      webVitals.onCLS(record, {reportAllChanges: true});
      webVitals.onINP(record, {reportAllChanges: true, durationThreshold: 16});
      webVitals.onLCP(record, {reportAllChanges: true});
    `,
  });
  const otherOrigins = new Set<string>();
  const origin = new URL(server.url).origin;
  page.on('request', (request) => {
    const url = new URL(request.url());
    if (['http:', 'https:'].includes(url.protocol) && url.origin !== origin)
      otherOrigins.add(url.origin);
  });
  await connectBrowser(page, server);
  await create(page);
  await input(page).fill('Input responsiveness fixture');
  await page.getByRole('tab', { name: '파일', exact: true }).click();
  await page.getByRole('button', { name: 'sample.go', exact: true }).click();
  await page.getByRole('tab', { name: '대화', exact: true }).click();
  await expect(input(page)).toHaveText('Input responsiveness fixture');
  const readMetrics = () =>
    page.evaluate(
      () => Reflect.get(window, '__mecaVitals') as Partial<Record<Metric['name'], number>>
    );
  await expect.poll(readMetrics, { message: 'Chromium reported CLS, INP and LCP' }).toMatchObject({
    CLS: expect.any(Number),
    INP: expect.any(Number),
    LCP: expect.any(Number),
  });
  const metrics = await readMetrics();
  await info.attach('local-web-vitals', {
    body: JSON.stringify(metrics),
    contentType: 'application/json',
  });
  expect(metrics.CLS).toBeLessThan(0.1);
  expect(metrics.INP).toBeLessThan(250);
  expect(metrics.LCP).toBeGreaterThan(0);
  expect([...otherOrigins]).toEqual([]);
});
