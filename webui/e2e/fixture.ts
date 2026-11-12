// Purpose: Share isolated server discovery and cookie authentication across browser behavior tests.
// Pattern: Browser test I/O fixture without bearer credentials in CLI arguments or traces.
// Usage: Feature specs read the descriptor once and authenticate their independent browser context.
// Related: cmd/meca/ui_browser_test.go, webui/e2e/editor-scroll.spec.ts.
import { readFile } from 'node:fs/promises';
import { test as base, expect, type Page } from '@playwright/test';

export type BrowserFixture = {
  url: string;
  tokenFile: string;
  projectA: string;
  projectB: string;
  home: string;
  provider: string;
  archiveSession: string;
};

export async function browserFixture(): Promise<BrowserFixture> {
  const path = process.env.MECA_UI_DESCRIPTOR;
  test.skip(!path, 'start the isolated Go fixture first');
  return JSON.parse(await readFile(path!, 'utf8'));
}

export async function connectBrowser(page: Page, fixture: BrowserFixture) {
  // Non-desktop behavior specs must not open a system dialog on the developer machine.
  await page.route('**/v1/ui/pick-folder', (route) => route.fulfill({ status: 503 }));
  const response = await page.request.post(`${fixture.url}/v1/ui/login`, {
    headers: {
      'X-Meca-UI': '1',
      Origin: fixture.url,
      Authorization: `Bearer ${(await readFile(fixture.tokenFile, 'utf8')).trim()}`,
    },
  });
  expect(response.ok(), 'isolated browser authentication').toBe(true);
  const language = await page.request.post(`${fixture.url}/v1/command`, {
    headers: { 'X-Meca-UI': '1', Origin: fixture.url },
    data: { setSettings: { language: 'ko' } },
  });
  expect(language.ok()).toBe(true);
  await page.goto(`${fixture.url}/ui/`);
  await expect(page.getByRole('tab', { name: '대화', exact: true })).toBeVisible();
}

export const input = (page: Page) => page.getByLabel('작업 입력', { exact: true });
export function historyPage(items: import('@/shared/api').HistoryItemJson[], before = '0') {
  let through = 0n;
  for (const item of items)
    if (BigInt(item.sequence ?? '0') > through) through = BigInt(item.sequence ?? '0');
  return { items, before, through: String(through) };
}
export async function create(page: Page, workspace = 'alpha') {
  const group = page
    .locator('.workspace-group')
    .filter({ has: page.locator('.workspace-name', { hasText: workspace }) });
  const heading = group.locator('.workspace-name');
  await heading.click();
  const previous = page.url();
  await group.getByRole('button', { name: '새 대화', exact: true }).click();
  await page.waitForURL((url) => url.href !== previous && url.pathname.includes('/c/session_'));
  await expect(input(page)).toBeEnabled();
  return page.url();
}
export async function send(page: Page, text: string) {
  await input(page).fill(text);
  await page.getByRole('button', { name: /^(전송|보내기)$/, exact: true }).click();
}
export async function rpc(
  page: Page,
  command: import('@/shared/api').CommandJson,
  workspace = ''
): Promise<import('@/shared/api').ReplyJson> {
  return page.evaluate(
    async ({ command, workspace }) => {
      const response = await fetch('/v1/command', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-Meca-UI': '1' },
        body: JSON.stringify({ ...command, workspaceId: workspace }),
      });
      const reply = await response.json();
      if (reply.error) throw Error(`${reply.error.code}: ${reply.error.message}`);
      return reply;
    },
    { command, workspace }
  );
}

export function required<T>(value: T | null | undefined): T {
  if (value === null || value === undefined) throw new Error('expected visible browser geometry');
  return value;
}

// Each test owns its cookie lifetime, including assertion failures and early exits.
export const test = base.extend<{ browserConnection: undefined }>({
  browserConnection: [
    async ({ page }, use) => {
      await use(undefined);
      const current = new URL(page.url());
      if (!['http:', 'https:'].includes(current.protocol)) return;
      const cookies = await page.context().cookies(`${current.origin}/v1/ui/session`);
      if (!cookies.some((cookie) => cookie.name.startsWith('meca_ui_'))) return;
      const closed = await page.request.post(`${current.origin}/v1/ui/logout`, {
        headers: { Origin: current.origin, 'X-Meca-UI': '1' },
      });
      expect(closed.status(), 'browser fixture revokes its own cookie').toBe(204);
    },
    { auto: true },
  ],
});
