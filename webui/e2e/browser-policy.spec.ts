// Purpose: Check UI document policies and preserve local drafts across authentication expiry.
// Pattern: Browser security and lifetime contracts without collecting report content.
// Usage: MECA_UI_DESCRIPTOR points to the isolated Go browser fixture.
// Related: internal/webui/headers.go, webui/src/pages/workspace/model/file-document.ts.
import { readFile } from 'node:fs/promises';
import { expect } from '@playwright/test';
import { browserFixture, connectBrowser, create, input, test } from '#e2e/fixture';

test('document policies retain clipboard support and render without Trusted Types violations', async ({
  page,
}) => {
  const fixture = await browserFixture();
  await page.addInitScript(() => {
    const reports: string[] = [];
    Object.assign(window, { policyDirectives: reports });
    document.addEventListener('securitypolicyviolation', (event) => {
      reports.push(`${event.disposition}:${event.effectiveDirective}`);
    });
  });
  await connectBrowser(page, fixture);
  const response = await page.request.get(`${fixture.url}/ui/`);
  const headers = response.headers();
  expect(headers['content-security-policy']).toContain("object-src 'none'");
  expect(headers['content-security-policy-report-only']).toBe("require-trusted-types-for 'script'");
  expect(headers['permissions-policy']).toContain('clipboard-read=(self)');
  expect(headers['cross-origin-resource-policy']).toBe('same-origin');
  const favicon = await page.request.get(`${fixture.url}/ui/favicon.svg`);
  expect(favicon.headers()['cache-control']).toContain('no-cache');
  await page.getByRole('tab', { name: '파일', exact: true }).click();
  await page.getByRole('button', { name: 'sample.go', exact: true }).click();
  await expect(page.locator('.text-editor textarea')).toBeVisible();
  await page.emulateMedia({ forcedColors: 'active', reducedMotion: 'reduce' });
  await expect(input(page)).toBeVisible();
  const reports = await page.evaluate(() => Reflect.get(window, 'policyDirectives') as string[]);
  expect(reports).toEqual([]);
});

test('authentication expiry closes sensitive settings and preserves composer and editor drafts', async ({
  page,
}) => {
  const fixture = await browserFixture();
  await connectBrowser(page, fixture);
  await create(page);
  await input(page).fill('Unsent local draft');
  await page.getByRole('tab', { name: '파일', exact: true }).click();
  await page.getByRole('button', { name: 'sample.go', exact: true }).click();
  const editor = page.locator('.text-editor textarea');
  await expect(editor).toBeVisible();
  await editor.fill('package unsaved_draft\n');
  await page.getByRole('button', { name: '설정', exact: true }).click();
  await expect(page.getByRole('dialog')).toBeVisible();
  const logout = await page.request.post(`${fixture.url}/v1/ui/logout`, {
    headers: { Origin: fixture.url, 'X-Meca-UI': '1' },
  });
  expect(logout.status()).toBe(204);
  const token = page.getByRole('textbox', { name: /^서버 토큰/ });
  await expect(token).toBeVisible();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  await token.fill((await readFile(fixture.tokenFile, 'utf8')).trim());
  await page.getByRole('button', { name: '연결', exact: true }).click();
  await expect(input(page)).toBeVisible();
  await expect(input(page)).toHaveText('Unsent local draft');
  await expect(editor).toHaveValue('package unsaved_draft\n');
  await expect(page.getByRole('button', { name: '저장', exact: true })).toBeEnabled();
});
