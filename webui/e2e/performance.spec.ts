// Purpose: Verify bounded editor/navigation DOM and one scoped refresh per short task burst.
// Pattern: Embedded-browser performance contracts using the isolated free Go fixture.
// Usage: MECA_UI_DESCRIPTOR=/tmp/task/connection.json bun run test:e2e.
// Related: cmd/meca/ui_browser_test.go, webui/src/pages/workspace/ui/navigation/session-list.tsx.
import { readFile, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { expect, test, type Page, type Response } from '@playwright/test';

let fixture: { url: string; tokenFile: string; projectA: string };
test.beforeAll(async () => {
  test.skip(!process.env.MECA_UI_DESCRIPTOR, 'start the isolated Go UI fixture first');
  fixture = JSON.parse(await readFile(process.env.MECA_UI_DESCRIPTOR!, 'utf8'));
});
test.beforeEach(async ({ page }) => {
  await page.goto(`${fixture.url}/ui/`);
  await page.getByRole('textbox', { name: '서버 토큰', exact: true }).fill((await readFile(fixture.tokenFile, 'utf8')).trim());
  await page.getByRole('button', { name: '연결', exact: true }).click();
  await page.locator('.workspace-name').filter({ hasText: 'alpha' }).waitFor();
});
const operation = (response: Response, key: string) => response.url().endsWith('/v1/command') && !!response.request().postDataJSON()?.[key];
async function create(page: Page) {
  await page.locator('.workspace-name').filter({ hasText: 'alpha' }).click();
  const sessions = page.waitForResponse(response => operation(response, 'listSessions'));
  const workspaces = page.waitForResponse(response => operation(response, 'listWorkspaces'));
  await page.locator('.workspace-group').filter({ has: page.locator('.workspace-name', { hasText: 'alpha' }) }).getByRole('button', { name: '새 대화', exact: true }).click();
  await expect(page.getByRole('heading', { name: '어떤 작업을 시작할까요?', exact: true })).toBeVisible();
  await Promise.all([(await sessions).finished(), (await workspaces).finished()]);
}

test('many-line files keep preview and editor DOM bounded and fetch the initial directory once', async ({ page }) => {
  await writeFile(join(fixture.projectA, 'many-lines.ts'), Array.from({ length: 10000 }, (_, index) => `// line ${index}`).join('\n'));
  await create(page);
  let directories = 0;
  page.on('request', request => {
    if (request.url().endsWith('/v1/command') && request.postDataJSON()?.directory) directories++;
  });
  await page.getByRole('tab', { name: '파일', exact: true }).click();
  await page.getByRole('button', { name: 'many-lines.ts', exact: true }).click();
  await page.getByRole('button', { name: '파일 편집', exact: true }).waitFor();
  expect(await page.locator('.file-document').locator('*').count()).toBeLessThan(2000);
  expect(directories).toBe(1);
  await page.getByRole('button', { name: '파일 편집', exact: true }).click();
  await expect(page.getByRole('textbox', { name: /many-lines.ts$/ })).toContainText('line 9999');
  expect(await page.locator('.text-editor').locator('*').count()).toBeLessThan(32);
});

test('conversation paging bounds DOM and evicted conversations retain their default model', async ({ page }) => {
  await create(page);
  const cold = page.url();
  await page.evaluate(async () => {
    const invoke = async (operation: object) => {
      const response = await fetch('/v1/command', { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-Meca-UI': '1' }, body: JSON.stringify(operation) });
      const reply = await response.json(); if (reply.error) throw Error(reply.error.code); return reply;
    };
    const workspaceId = (await invoke({ listWorkspaces: {} })).workspaces.items.find((value: { name: string }) => value.name === 'alpha').id;
    for (let index = 0; index < 90; index++) await invoke({ workspaceId, selectSession: { create: true } });
  });
  const group = page.locator('.workspace-group').filter({ has: page.locator('.workspace-name', { hasText: 'alpha' }) });
  await group.getByRole('button', { name: '이전 이력 더 보기', exact: true }).waitFor();
  expect(await page.locator('.session-row').count()).toBeLessThan(25);
  const next = page.waitForResponse(response => operation(response, 'listSessions') && !!response.request().postDataJSON()?.listSessions.cursor);
  await group.getByRole('button', { name: '이전 이력 더 보기', exact: true }).click();
  expect((await (await next).json()).error).toBeUndefined();
  expect(await page.locator('.session-row').count()).toBeLessThan(25);
  await page.goto(cold);
  await page.getByLabel('작업 입력', { exact: true }).fill('Reopened after runtime eviction');
  await page.getByRole('button', { name: /^(전송|보내기)$/, exact: true }).click();
  await expect(page.locator('.history-rows')).toContainText('Completed: Reopened after runtime eviction');
});

test('a short task avoids repeated state and metadata refreshes', async ({ page }) => {
  await create(page);
  const counts: Record<string, number> = {};
  page.on('request', request => {
    if (!request.url().endsWith('/v1/command')) return;
    const key = Object.keys(request.postDataJSON()).find(value => !['id', 'workspaceId'].includes(value));
    if (key) counts[key] = (counts[key] ?? 0) + 1;
  });
  const sessions = page.waitForResponse(response => operation(response, 'listSessions'));
  const workspaces = page.waitForResponse(response => operation(response, 'listWorkspaces'));
  await page.getByLabel('작업 입력', { exact: true }).fill('Performance burst contract');
  await page.getByRole('button', { name: /^(전송|보내기)$/, exact: true }).click();
  await expect(page.locator('.history-rows')).toContainText('Completed: Performance burst contract');
  await Promise.all([(await sessions).finished(), (await workspaces).finished()]);
  expect(counts.start).toBe(1);
  // Lazy model preparation emits one model change; text and completion reuse event facts.
  expect(counts.state ?? 0).toBeLessThanOrEqual(1);
  expect(Object.values(counts).reduce((sum, value) => sum + value, 0)).toBeLessThanOrEqual(8);
});
