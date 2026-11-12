// Purpose: Keep failed completion metadata independent of mixed response payloads in a real browser.
// Pattern: Isolated server fixture with a deliberately malformed read response.
// Usage: MECA_UI_DESCRIPTOR=... bun run test:e2e e2e/completion-boundary.spec.ts.
// Related: webui/src/pages/workspace/ui/chat/completion-row.tsx, cmd/meca/ui_browser_test.go.

import { expect } from '@playwright/test';
import { browserFixture, connectBrowser, required, test } from '#e2e/fixture';

test('completion metadata cannot render response, thinking or invalid numeric strings', async ({
  page,
}) => {
  const fixture = await browserFixture();
  const reason = 'Terminal reason explicitly requested.';
  let append = false;
  await page.route('**/v1/command', async (route) => {
    const command = route.request().postDataJSON();
    if (command.detail?.historySequence === '2') {
      await route.fulfill({
        json: {
          detail: {
            text: reason,
            totalBytes: String(reason.length),
            nextOffset: String(reason.length),
            revision: '1'.repeat(64),
            contentType: 'text/plain',
          },
        },
      });
      return;
    }
    if (!command.history) return route.continue();
    await route.fulfill({
      json: {
        history: {
          items: [
            { sequence: '1', kind: 'message', role: 'assistant', text: 'The actual response.' },
            {
              sequence: '2',
              kind: 'completion',
              role: 'assistant',
              text: 'LEAKED_AGENT_TEXT',
              event: { kind: 'progress', text: 'LEAKED_EVENT_TEXT', thinking: 'LEAKED_THINKING' },
              completion: {
                status: 'FAILED',
                statistics: {
                  elapsedMillis: '9000',
                  generationMillis: '3000',
                  usage: { input: 'LEAKED_TOKEN_TEXT', output: '30' },
                },
              },
            },
            ...(append
              ? [{ sequence: '3', kind: 'message', role: 'user', text: 'A later message.' }]
              : []),
          ],
        },
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
  await expect(page.getByText(/LEAKED_/)).toHaveCount(0);
  const answer = await page
    .getByText('The actual response.', { exact: true })
    .boundingBox()
    .then(required);
  const summary = await completion.boundingBox().then(required);
  expect(answer.y + answer.height <= summary.y).toBeTruthy();
  await completion.getByRole('button').click();
  await expect(page.locator('.original-text')).toContainText(reason);
  expect(
    await page.locator('.original-modal').evaluate((node) => getComputedStyle(node).backgroundColor)
  ).toMatch(/^rgb\(/);
  await expect(completion).not.toContainText(reason);
  append = true;
  const parts = new URL(page.url()).pathname.split('/');
  const started = await page.request.post(`${fixture.url}/v1/command`, {
    headers: { 'X-Meca-UI': '1', Origin: fixture.url },
    data: {
      id: 'completion-inspection-lifetime',
      workspaceId: parts[parts.indexOf('w') + 1],
      start: {
        sessionId: parts[parts.indexOf('c') + 1],
        key: 'completion-inspection-lifetime',
        text: 'Verify inspector lifetime.',
      },
    },
  });
  expect((await started.json()).task?.id).toBeTruthy();
  await expect(page.getByText('A later message.', { exact: true })).toBeVisible();
  await expect(page.locator('.original-text')).toContainText(reason);
});
