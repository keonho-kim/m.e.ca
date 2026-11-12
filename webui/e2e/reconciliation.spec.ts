// Purpose: Verify delayed filesystem replies cannot race submissions and reconnects restore missed usage.
// Pattern: Browser-to-engine synchronization regression tests.
// Usage: MECA_UI_DESCRIPTOR=/tmp/task/connection.json bun run test:e2e.
// Related: webui/src/pages/workspace/model/file-references.ts, webui/src/pages/workspace/api/events.ts.

import { mkdir, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { expect, type Page } from '@playwright/test';
import { type BrowserFixture, browserFixture, connectBrowser, test } from '#e2e/fixture';

let fixture: BrowserFixture;
type Reply = { task?: { id: string; status: string }; error?: { code: string } };
const invoke = (page: Page, command: object) =>
  page.evaluate(async (value) => {
    const response = await fetch('/v1/command', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Meca-UI': '1' },
      body: JSON.stringify(value),
    });
    const reply = await response.json();
    if (reply.error) throw new Error(reply.error.code);
    return reply as Reply;
  }, command);
test.beforeAll(async () => {
  fixture = await browserFixture();
});
test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => {
    const streams: EventSource[] = [];
    Object.assign(window, { testStreams: streams });
    const Native = EventSource;
    window.EventSource = class extends Native {
      constructor(url: string | URL, options?: EventSourceInit) {
        super(url, options);
        streams.push(this);
      }
    };
  });
  await connectBrowser(page, fixture);
  const previous = page.url();
  await page
    .locator('.workspace-group')
    .filter({ has: page.locator('.workspace-name', { hasText: 'alpha' }) })
    .getByRole('button', { name: '새 대화', exact: true })
    .click();
  await page.waitForURL((url) => url.href !== previous && url.pathname.includes('/c/session_'));
  await expect(page.locator('.execution-status')).toContainText(/입력 0.*출력 0/);
});
const target = (page: Page) => {
  const parts = new URL(page.url()).pathname.split('/');
  return { workspaceId: parts[parts.indexOf('w') + 1], sessionId: parts.at(-1) };
};

test('one batch rename reconciles 24 references before a queued send can start', async ({
  page,
}) => {
  const suffix = crypto.randomUUID().slice(0, 8);
  const original = `batch-source-${suffix}`,
    renamed = `batch-renamed-${suffix}`;
  await mkdir(join(fixture.projectA, original));
  const selected = Array.from({ length: 24 }, (_, index) => `${original}/ref-${index}.txt`);
  const owner = target(page);
  for (const path of selected) {
    await writeFile(join(fixture.projectA, path), 'Reference fixture');
    await invoke(page, {
      workspaceId: owner.workspaceId,
      selectFile: { sessionId: owner.sessionId, path, selected: true },
    });
  }
  await page.reload();
  await expect(page.locator('.composer')).toContainText(selected[23]!);
  await page.getByRole('tab', { name: '파일', exact: true }).click();
  let entered!: () => void, release!: () => void;
  const applied = new Promise<void>((resolve) => {
    entered = resolve;
  });
  const delayed = new Promise<void>((resolve) => {
    release = resolve;
  });
  const counts = { mutate: 0, select: 0, selections: 0, starts: 0 };
  const submitted: string[][] = [];
  await page.route('**/v1/command', async (route) => {
    const command = route.request().postDataJSON();
    if (command.selectFile) counts.select++;
    if (command.fileSelections) counts.selections++;
    if (command.start) {
      counts.starts++;
      submitted.push(command.start.paths);
    }
    if (!command.mutateFile) {
      await route.continue();
      return;
    }
    counts.mutate++;
    const reply = await route.fetch();
    entered();
    await delayed;
    await route.fulfill({ response: reply });
  });
  try {
    await page.getByRole('button', { name: `파일 작업 · ${original}`, exact: true }).click();
    await page.getByRole('menuitem', { name: '이름 바꾸기', exact: true }).click();
    const dialog = page.getByRole('dialog', { name: '이름 바꾸기', exact: true });
    await dialog.getByRole('textbox', { name: '이름', exact: true }).fill(renamed);
    await dialog.getByRole('button', { name: '저장', exact: true }).click();
    await applied;
    await page.getByLabel('작업 입력', { exact: true }).fill('METRICS_FIXTURE delayed file rename');
    const send = page.getByRole('button', { name: /^(전송|보내기)$/ });
    await send.click();
    await expect(send).toBeDisabled();
    expect(counts.starts).toBe(0);
    release();
    await expect.poll(() => counts.starts).toBe(1);
    expect(submitted[0]?.length).toBe(24);
    expect(submitted[0]?.every((path) => path.startsWith(`${renamed}/`))).toBe(true);
    expect(counts).toEqual({ mutate: 1, select: 0, selections: 0, starts: 1 });
  } finally {
    release();
    await page.unroute('**/v1/command');
  }
});

test('returning to a conversation refreshes usage missed while its EventSource was closed', async ({
  page,
}) => {
  const owner = target(page);
  await page.evaluate(() => {
    for (const stream of Reflect.get(window, 'testStreams') as EventSource[]) stream.close();
  });
  const prompt = `METRICS_FIXTURE reconnect-${crypto.randomUUID()}`;
  const started = await invoke(page, {
    workspaceId: owner.workspaceId,
    start: { sessionId: owner.sessionId, key: crypto.randomUUID(), text: prompt },
  });
  await expect
    .poll(
      async () =>
        (
          await invoke(page, {
            workspaceId: owner.workspaceId,
            taskStatus: { taskId: started.task?.id },
          })
        ).task?.status
    )
    .toBe('answered');
  await expect(page.locator('.execution-status')).toContainText(/입력 0.*출력 0/);
  await page
    .locator('.workspace-group')
    .filter({ has: page.locator('.workspace-name', { hasText: 'alpha' }) })
    .getByRole('button', { name: '새 대화', exact: true })
    .click();
  await page.locator('.session-row').filter({ hasText: prompt }).click();
  await expect(page.locator('.execution-status')).toContainText(/입력 13.*출력 7/);
  await expect(page.locator('.completion-summary')).toContainText(/입력 13.*출력 7/);
});
