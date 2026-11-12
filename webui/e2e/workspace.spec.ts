// Purpose: Verify independent workspace execution, queued follow-ups and durable conversation history.
// Pattern: Browser-to-engine contracts with an isolated provider and owned hold cleanup.
// Usage: MECA_UI_DESCRIPTOR points to the opt-in Go browser fixture.
// Related: webui/e2e/fixture.ts, cmd/meca/ui_browser_test.go.
import { expect } from '@playwright/test';
import {
  type BrowserFixture,
  browserFixture,
  connectBrowser,
  create,
  input,
  rpc,
  send,
  test,
} from '#e2e/fixture';

let fixture: BrowserFixture;
test.beforeAll(async () => {
  fixture = await browserFixture();
});
test.beforeEach(async ({ page }) => {
  await connectBrowser(page, fixture);
});
test.afterEach(async ({ page }) => {
  await page.request.post(`${fixture.provider}/test/release`);
});
test('multiple projects continue independently and one completed predecessor runs a follow-up once', async ({
  page,
}) => {
  const first = await create(page);
  await send(page, 'HOLD_FIXTURE alpha first');
  await expect(page.getByRole('button', { name: '작업 중지', exact: true })).toBeVisible();
  await input(page).fill('Captured follow-up once');
  for (const modifier of ['ctrlKey', 'metaKey']) {
    const allowed = await input(page).evaluate(
      (node, key) =>
        node.dispatchEvent(
          new KeyboardEvent('keydown', { key: 'Tab', [key]: true, bubbles: true, cancelable: true })
        ),
      modifier
    );
    expect(allowed).toBe(true);
  }
  await input(page).press('Control+Enter');
  await expect(page.getByRole('region', { name: '예약 요청' })).toBeVisible();
  await create(page);
  await send(page, 'HOLD_FIXTURE alpha second');
  await expect(page.getByRole('button', { name: '작업 중지', exact: true })).toBeVisible();
  await create(page, 'beta');
  await send(page, 'HOLD_FIXTURE beta first');
  await expect(page.getByRole('button', { name: '작업 중지', exact: true })).toBeVisible();
  expect(
    (await rpc(page, { listWorkspaces: {} })).workspaces?.items?.reduce(
      (n, w) => n + (w.activeTasks ?? 0),
      0
    )
  ).toBe(3);
  await expect(page.locator('.live-output')).toContainText('Fixture');
  await page.getByRole('button', { name: '작업 중지', exact: true }).click();
  await expect(page.getByRole('button', { name: '작업 중지', exact: true })).toBeHidden();
  await page.request.post(`${fixture.provider}/test/release`);
  await page.locator('.session-link').filter({ hasText: 'HOLD_FIXTURE alpha first' }).click();
  await expect(page.locator('.history-rows')).toContainText('Captured follow-up once');
  const parts = new URL(first).pathname.split('/');
  await expect
    .poll(
      async () =>
        (await rpc(page, { listTasks: { sessionId: parts[5] } }, parts[3])).tasks?.items?.length
    )
    .toBe(2);
  await expect(page.getByRole('region', { name: '예약 요청' })).toBeHidden();
});

test('long history keeps a bounded page window and can return to the latest saved response', async ({
  page,
}) => {
  const workspaces = (await rpc(page, { listWorkspaces: {} })).workspaces!;
  await page.goto(`${fixture.url}/ui/w/${workspaces.defaultId}/c/${fixture.archiveSession}`);
  await expect(page.locator('.history-rows')).toContainText('Archive message 299');
  for (let index = 0; index < 4; index++) {
    const response = page.waitForResponse(
      (response) =>
        response.url().endsWith('/v1/command') && !!response.request().postDataJSON()?.history
    );
    await page
      .getByRole('tabpanel', { name: '대화', exact: true })
      .getByRole('button', { name: '이전 이력 더 보기', exact: true })
      .click();
    await response;
  }
  await page.getByRole('button', { name: '최근 대화로 돌아가기', exact: true }).click();
  await expect(page.locator('.history-rows')).toContainText('Archive message 299');
  expect(await page.locator('.history-row').count()).toBeLessThan(50);
});

test('deleting an idle conversation removes its saved content and browser navigation cannot restore it', async ({
  page,
}) => {
  await create(page);
  await send(page, 'Delete this fixture conversation');
  await expect(page.locator('.history-rows')).toContainText(
    'Completed: Delete this fixture conversation'
  );
  await expect(page.getByRole('button', { name: '작업 중지', exact: true })).toBeHidden();
  await page
    .locator('.session-row[aria-current=page]')
    .locator('..')
    .getByRole('button', { name: '대화 삭제', exact: true })
    .click();
  await page
    .getByRole('dialog', { name: '대화 삭제', exact: true })
    .getByRole('button', { name: '대화 삭제', exact: true })
    .click();
  await expect(page).not.toHaveURL(/\/c\/session_/);
  await page.goBack();
  await expect(
    page.locator('.history-rows').filter({ hasText: 'Delete this fixture conversation' })
  ).toHaveCount(0);
});
