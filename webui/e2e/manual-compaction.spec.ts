// Purpose: Verify a manual compaction command submits a summary-only task for the open conversation.
// Pattern: Browser workflow test over the real native API.
// Usage: Run through the Playwright browser suite.
// Related: webui/src/pages/workspace/model/commands.ts, webui/e2e/fixture.ts.
import { expect } from '@playwright/test';
import { connectBrowser, create, input, send, test } from '#e2e/fixture';

test('manual compaction keeps the short conversation visible after reload', async ({
  page,
  server: fixture,
}) => {
  await connectBrowser(page, fixture);
  await create(page);
  const original = 'Remember that the selected folder is blue.';
  await send(page, original);
  await expect(page.getByText(`Completed: ${original}`, { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: '중지', exact: true })).toHaveCount(0);

  const submitted = page.waitForRequest(
    (request) =>
      request.method() === 'POST' &&
      request.url().endsWith('/v1/command') &&
      request.postDataJSON()?.start?.text === '!compact'
  );
  await input(page).fill('!compact');
  await input(page).press('Escape');
  await input(page).press('Enter');
  const command = (await submitted).postDataJSON().start;
  expect(command.sessionId).toMatch(/^session_/);
  expect(command.attachmentIds ?? []).toHaveLength(0);
  expect(command.paths ?? []).toHaveLength(0);
  await expect(page.getByText('문맥 압축됨', { exact: true })).toHaveCount(1);
  await expect(input(page)).toHaveText('');

  await page.reload();
  const history = page.getByRole('log');
  await expect(history).toHaveAttribute('aria-live', 'polite');
  await expect(history.getByText(original, { exact: true })).toBeVisible();
  await expect(history.getByText('문맥 압축됨', { exact: true })).toHaveCount(1);
});
