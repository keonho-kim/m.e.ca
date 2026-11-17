// Purpose: Verify stream-follow controls, chronological recovery and concise approvals in the browser.
// Pattern: Controlled native fixture and explicit snapshot rendering contracts.
// Usage: MECA_UI_DESCRIPTOR points to the isolated embedded server.
// Related: cmd/meca/ui_burst_provider_test.go, webui/src/pages/workspace/model/live-timeline.ts.
import { expect } from '@playwright/test';
import { browserFixture, connectBrowser, create, rpc, send, test } from '#e2e/fixture';

test.afterEach(async ({ page }) => {
  const parts = new URL(page.url()).pathname.split('/');
  if (!parts[3] || !parts[5]) return;
  await page.unroute('**/v1/command');
  const fixture = await browserFixture();
  const tasks =
    (await rpc(page, { listTasks: { sessionId: parts[5] } }, parts[3])).tasks?.items ?? [];
  for (const task of tasks) {
    if (!task.id || ['answered', 'stopped', 'interrupted'].includes(task.status ?? '')) continue;
    await page.request.post(`${fixture.url}/v1/workspaces/${parts[3]}/tasks/${task.id}/cancel`, {
      headers: { 'X-Meca-UI': '1', Origin: fixture.url },
    });
  }
});

test('automatic streaming never flashes the jump button, while reading history exposes it', async ({
  page,
}) => {
  const fixture = await browserFixture();
  await connectBrowser(page, fixture);
  await page.emulateMedia({ reducedMotion: 'no-preference' });
  await create(page);
  await send(page, 'SCROLL_BURST_FIXTURE');
  await expect(page.locator('.live-output')).toContainText('Burst ready.');
  const button = page.locator('.astryx-chat-layout-scroll-button');
  await expect(button).toHaveAttribute('inert', '');
  await button.evaluate((node) => {
    const owner = window as typeof window & { jumpFlashes: number; jumpObserver: MutationObserver };
    owner.jumpFlashes = 0;
    owner.jumpObserver = new MutationObserver((changes) => {
      for (const change of changes)
        if (change.attributeName === 'inert' && change.oldValue !== null) owner.jumpFlashes++;
    });
    owner.jumpObserver.observe(node, {
      attributes: true,
      attributeFilter: ['inert'],
      attributeOldValue: true,
    });
  });
  await page.request.post(`${fixture.provider}/test/start-burst`);
  await expect(page.getByRole('button', { name: '작업 중지', exact: true })).toBeHidden();
  const scroller = page.locator('.workspace-body');
  await expect
    .poll(() => scroller.evaluate((node) => node.scrollHeight - node.clientHeight - node.scrollTop))
    .toBeLessThan(30);
  const flashes = await page.evaluate(() => {
    const owner = window as typeof window & { jumpFlashes: number; jumpObserver: MutationObserver };
    owner.jumpObserver.disconnect();
    return owner.jumpFlashes;
  });
  expect(flashes).toBe(0);
  await scroller.evaluate((node) => {
    node.scrollTop -= 800;
  });
  await expect(button).not.toHaveAttribute('inert');
  await button.getByRole('button').click();
  await expect(button).toHaveAttribute('inert', '');
  await expect
    .poll(() => scroller.evaluate((node) => node.scrollHeight - node.clientHeight - node.scrollTop))
    .toBeLessThan(30);
});

test('snapshot tool and skill entries stay between their neighboring generations', async ({
  page,
}) => {
  const fixture = await browserFixture();
  await connectBrowser(page, fixture);
  await create(page);
  await send(page, 'HOLD_FIXTURE ordered recovery');
  await expect(page.getByRole('button', { name: '작업 중지', exact: true })).toBeVisible();
  await page.route('**/v1/command', async (route) => {
    const command = route.request().postDataJSON();
    if (!command.state?.sessionId) return route.continue();
    const response = await route.fetch();
    const reply = await response.json();
    const snapshot = reply.state;
    const taskId = snapshot.task.id;
    const base = {
      taskId,
      workspaceId: command.workspaceId,
      sessionId: command.state.sessionId,
      agent: 'z',
      runId: 'ordered',
    };
    snapshot.drafts = [
      {
        taskId,
        agent: 'a',
        generation: '2',
        startSequence: '9',
        thinking: 'Later reasoning after skill.',
      },
      {
        taskId,
        agent: 'z',
        generation: '1',
        startSequence: '3',
        thinking: 'Earlier reasoning before skill.',
      },
    ];
    snapshot.activity = [
      {
        ...base,
        sequence: '7',
        kind: 'skill_loaded',
        notice: { code: 'SKILL_READ', skill: { name: 'ordered-skill' } },
      },
      {
        ...base,
        sequence: '5',
        kind: 'tool_started',
        tool: { executionId: 'ordered-call', name: 'load_skill', phase: 'TOOL_PHASE_RUNNING' },
      },
    ];
    await route.fulfill({ response, json: reply });
  });
  for (let count = 0; count < 2; count++) {
    await page.reload();
    const output = page.locator('.live-output');
    await expect(output).toContainText('Later reasoning after skill.');
    const text = (await output.textContent()) ?? '';
    expect(text.indexOf('Earlier reasoning')).toBeLessThan(text.indexOf('load_skill'));
    expect(text.indexOf('load_skill')).toBeLessThan(text.indexOf('ordered-skill'));
    expect(text.indexOf('ordered-skill')).toBeLessThan(text.indexOf('Later reasoning'));
  }
  await page.unroute('**/v1/command');
  await page.getByRole('button', { name: '작업 중지', exact: true }).click();
});

test('approval shows reasons first and fetches the original only on request', async ({ page }) => {
  await connectBrowser(page, await browserFixture());
  await create(page);
  let reads = 0;
  await page.route('**/v1/command', async (route) => {
    if (route.request().postDataJSON().detail?.approvalId) reads++;
    await route.continue();
  });
  await send(page, 'APPROVAL_FIXTURE');
  const modal = page.getByRole('dialog', { name: '승인 요청', exact: true });
  await expect(modal).toBeVisible();
  await expect(modal).toContainText('Mutation outside the project requires approval');
  await expect(modal).not.toContainText('approved fixture');
  expect(reads).toBe(0);
  await modal.getByRole('button', { name: '원문 보기', exact: true }).click();
  const detail = page.getByRole('dialog', { name: '상세 보기', exact: true });
  await expect(detail).toContainText('approved fixture');
  expect(reads).toBeGreaterThan(0);
  await detail.getByRole('button', { name: '닫기', exact: true }).click();
  await expect(modal).toBeVisible();
  await modal.getByRole('button', { name: '거절', exact: true }).click();
  await expect(modal).toBeHidden();
});
