// Purpose: Keep terminal status and measurements visible without original-detail actions.
// Pattern: Isolated server fixture with a deliberately malformed read response.
// Usage: Run through the Playwright browser suite.
// Related: webui/src/pages/workspace/ui/chat/completion-row.tsx, cmd/meca/ui_browser_test.go.

import { expect } from '@playwright/test';
import { connectBrowser, create, historyPage, required, test } from '#e2e/fixture';

test('completion rows preserve status without detail actions through later updates', async ({
  server: fixture,
  page,
}) => {
  let status: 'FAILED' | 'INTERRUPTED' | 'COMPLETED' = 'INTERRUPTED';
  let detailReads = 0;
  let append = false;
  await page.route('**/v1/command', async (route) => {
    const command = route.request().postDataJSON();
    if (command.detail?.historySequence) detailReads++;
    if (!command.history) return route.continue();
    await route.fulfill({
      json: {
        history: historyPage([
          {
            sequence: '1',
            message: { role: 'assistant', text: 'The actual response.', incomplete: true },
          },
          {
            sequence: '2',
            completion: {
              status,
              statistics: {
                elapsedMillis: '9000',
                generationMillis: '3000',
                usage: { input: '32', output: '30' },
              },
            },
          },
          ...(append
            ? [{ sequence: '3', message: { role: 'user', text: 'A later message.' } }]
            : []),
        ]),
      },
    });
  });
  await connectBrowser(page, fixture);
  await page
    .locator('.workspace-group')
    .filter({ has: page.locator('.workspace-name', { hasText: 'alpha' }) })
    .getByRole('button', { name: '새 대화', exact: true })
    .click();
  await expect(page.getByText('The actual response.', { exact: true })).toBeVisible();
  const completion = page.locator('.completion-row');
  await expect(completion).toHaveCount(1);
  await expect(completion).toContainText('00:00:09');
  await expect(completion).toContainText('10.0 tok/s');
  const answer = await page
    .getByText('The actual response.', { exact: true })
    .boundingBox()
    .then(required);
  const summary = await completion.boundingBox().then(required);
  expect(answer.y + answer.height <= summary.y).toBeTruthy();
  await expect(completion.getByRole('button')).toHaveCount(0);
  await expect(completion).toContainText('중단됨');
  for (const next of ['FAILED', 'COMPLETED'] as const) {
    status = next;
    await page.reload();
    await expect(completion).toContainText('10.0 tok/s');
    await expect(completion.getByRole('button')).toHaveCount(0);
    if (next === 'FAILED') await expect(completion).toContainText('실패');
    else await expect(completion).not.toContainText(/중단됨|실패/);
  }
  append = true;
  const parts = new URL(page.url()).pathname.split('/');
  const started = await page.request.post(`${fixture.url}/v1/command`, {
    headers: { 'X-Meca-UI': '1', Origin: fixture.url },
    data: {
      id: 'completion-measurements-lifetime',
      workspaceId: parts[parts.indexOf('w') + 1],
      start: {
        sessionId: parts[parts.indexOf('c') + 1],
        key: 'completion-measurements-lifetime',
        text: 'Verify measurements lifetime.',
      },
    },
  });
  expect((await started.json()).task?.id).toBeTruthy();
  await expect(page.getByText('A later message.', { exact: true })).toBeVisible();
  await expect(completion.getByRole('button')).toHaveCount(0);
  expect(detailReads).toBe(0);
});

// Invalid transport content must never fall back to a system-message row.
test('mixed completion and authored text are rejected at the browser receive boundary', {
  tag: '@smoke',
}, async ({ server: fixture, page }) => {
  await page.route('**/v1/command', async (route) => {
    if (route.request().postDataJSON().history)
      await route.fulfill({
        json: {
          history: {
            through: '1',
            items: [
              {
                sequence: '1',
                completion: { status: 'COMPLETED' },
                message: { role: 'assistant', text: 'INVALID_BODY_CANARY' },
              },
            ],
          },
        },
      });
    else await route.continue();
  });
  await connectBrowser(page, fixture);
  await create(page);
  await expect(
    page.locator('.chat-content').getByRole('button', { name: /다시 확인/ })
  ).toBeVisible();
  await expect(page.getByText(/INVALID_BODY_CANARY/)).toHaveCount(0);
  await expect(page.locator('.completion-row')).toHaveCount(0);
});
