// Purpose: Keep the surviving peer visible when a selected peer leaves live output.
// Pattern: Browser regression with controlled reconnect snapshots.
// Usage: Run through the Playwright browser suite.
// Related: webui/src/pages/workspace/ui/chat/live-output.tsx.
import { expect } from '@playwright/test';
import { connectBrowser, create, send, test } from '#e2e/fixture';

test('selected peer retirement restores visible remaining output', async ({
  server: fixture,
  page,
}, info) => {
  await connectBrowser(page, fixture);
  await create(page);
  await send(page, 'HOLD_FIXTURE team filter');
  await expect(page.getByRole('button', { name: '작업 중지', exact: true })).toBeVisible();
  let retired = false;
  await page.route('**/v1/command', async (route) => {
    const command = route.request().postDataJSON();
    if (!command.state?.sessionId) return route.continue();
    const response = await route.fetch();
    const reply = await response.json();
    const taskId = reply.state.task.id;
    reply.state.drafts = [
      ...(!retired
        ? [{ taskId, agent: 'Hana', generation: '1', text: 'Hana live contribution' }]
        : []),
      { taskId, agent: 'Ren', generation: '2', text: 'Ren final answer remains visible' },
    ];
    await route.fulfill({ response, json: reply });
  });
  const reconnect = async () => {
    await page.evaluate(() => {
      window.dispatchEvent(new PageTransitionEvent('pagehide', { persisted: true }));
      window.dispatchEvent(new PageTransitionEvent('pageshow', { persisted: true }));
    });
  };
  await reconnect();
  const selector = page.getByRole('combobox', { name: '팀원', exact: true });
  await selector.click();
  await page.getByRole('option', { name: 'Hana', exact: true }).click();
  await expect(page.locator('.live-output')).toContainText('Hana live contribution');
  await expect(page.locator('.live-output')).not.toContainText('Ren final answer');
  retired = true;
  await reconnect();
  await expect(selector).toBeHidden();
  await expect(page.locator('.live-output')).toContainText('Ren final answer remains visible');
  await page.screenshot({ path: info.outputPath('surviving-peer.png'), fullPage: true });
  await page.unroute('**/v1/command');
  await page.getByRole('button', { name: '작업 중지', exact: true }).click();
});

test('reconnect displays the committed preview after draft memory is released', async ({
  server: fixture,
  page,
}) => {
  await connectBrowser(page, fixture);
  await create(page);
  await send(page, 'HOLD_FIXTURE committed reconnect');
  await expect(page.getByRole('button', { name: '작업 중지', exact: true })).toBeVisible();
  await page.route('**/v1/command', async (route) => {
    const command = route.request().postDataJSON();
    if (!command.state?.sessionId) return route.continue();
    const response = await route.fetch();
    const reply = await response.json();
    const taskId = reply.state.task.id;
    reply.state.drafts = [
      {
        taskId,
        agent: 'Ren',
        generation: '2',
        presentation: 'saved.00000000000000000001',
        text: '',
        committed: true,
        truncated: true,
        historySequence: '999999',
      },
    ];
    reply.state.activity = [
      {
        taskId,
        workspaceId: command.workspaceId,
        sessionId: command.state.sessionId,
        agent: 'Ren',
        kind: 'team_progress',
        presentation: 'saved.00000000000000000001',
        historySequence: '999999',
        message: { role: 'assistant', text: 'Saved response awaiting history' },
      },
    ];
    await route.fulfill({ response, json: reply });
  });
  await page.reload();
  await expect(page.locator('.live-output')).toContainText('Saved response awaiting history');
  await page.unroute('**/v1/command');
  await page.getByRole('button', { name: '작업 중지', exact: true }).click();
});
