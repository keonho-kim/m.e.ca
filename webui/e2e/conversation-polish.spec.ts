// Purpose: Verify stable chat cells, reference tokens, settings confirmation and wrapped content.
// Pattern: Browser-to-engine presentation regression contracts.
// Usage: MECA_UI_DESCRIPTOR=/tmp/task/connection.json bun run test:e2e conversation-polish.spec.ts.
// Related: cmd/meca/ui_provider_test.go, webui/src/pages/workspace/ui/chat/chat.tsx.

import { mkdir, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { expect, type Page } from '@playwright/test';
import { type BrowserFixture, browserFixture, connectBrowser, required, test } from '#e2e/fixture';

let fixture: BrowserFixture;
test.beforeAll(async () => {
  fixture = await browserFixture();
  await mkdir(join(fixture.projectA, 'polish-folder'), { recursive: true });
  await writeFile(
    join(fixture.projectA, 'polish-folder', 'wrap.md'),
    '# Wrapped\n\n' +
      'unbroken_word_'.repeat(250) +
      '\n\n```text\n' +
      'unbroken_code_'.repeat(250) +
      '\n```'
  );
});
test.beforeEach(async ({ page }) => {
  await connectBrowser(page, fixture);
  const previous = page.url();
  await page
    .locator('.workspace-group')
    .filter({ has: page.locator('.workspace-name', { hasText: 'alpha' }) })
    .getByRole('button', { name: '새 대화', exact: true })
    .click();
  await page.waitForURL((url) => url.href !== previous && url.pathname.includes('/c/session_'));
  await expect(page.locator('.execution-status')).toContainText('입력');
});
const input = (page: Page) => page.getByRole('combobox', { name: '작업 입력', exact: true });
async function send(page: Page, text: string) {
  await input(page).fill(text);
  await page.locator('.composer').getByRole('button', { name: '전송', exact: true }).click();
}
async function noOverflow(page: Page, selector: string) {
  return page.locator(selector).evaluateAll((nodes) =>
    nodes
      .flatMap((root) => [root, ...root.querySelectorAll('*')])
      .filter((node) => {
        if (!(node instanceof HTMLElement) || !node.getClientRects().length || node.clientWidth < 5)
          return false;
        return (
          node.scrollWidth > node.clientWidth + 2 &&
          ['auto', 'scroll'].includes(getComputedStyle(node).overflowX)
        );
      })
      .map((node) => ({
        tag: node.tagName,
        class: node.className,
        width: node.clientWidth,
        scroll: node.scrollWidth,
      }))
  );
}

test('references stay in their work area and sent messages show icon tokens', async ({ page }) => {
  await page.getByRole('tab', { name: '파일', exact: true }).click();
  for (const name of ['README.md', 'polish-folder']) {
    await page.getByRole('button', { name: `파일 작업 · ${name}`, exact: true }).click();
    await page.getByRole('menuitem', { name: '추가하기', exact: true }).click();
    await expect(page.getByRole('tab', { name: '파일', exact: true })).toHaveAttribute(
      'aria-selected',
      'true'
    );
  }
  await expect(page.locator('.composer .reference-token')).toHaveCount(2);
  await page.getByRole('tab', { name: '스킬', exact: true }).click();
  await page
    .locator('.skill-entry')
    .filter({ hasText: 'ui-design' })
    .getByRole('button', { name: /입력에 스킬 넣기/ })
    .click();
  await expect(page.getByRole('tab', { name: '스킬', exact: true })).toHaveAttribute(
    'aria-selected',
    'true'
  );
  await expect(input(page).locator('[data-astryx-token]')).toHaveCount(1);
  await input(page).pressSequentially(' Inspect these references');
  await page.locator('.composer').getByRole('button', { name: '전송', exact: true }).click();
  await expect(page.getByRole('tab', { name: '대화', exact: true })).toHaveAttribute(
    'aria-selected',
    'true'
  );
  const message = page.locator('.history-row.user-message').last();
  await expect(message).toContainText('Inspect these references');
  await expect(message).not.toContainText('Selected path references');
  await expect(message.locator('.reference-token')).toHaveCount(2);
  expect(await message.locator('svg').count()).toBeGreaterThanOrEqual(3);
  await expect(page.locator('.execution-status')).not.toContainText('대기 중');
});

test('save asks for confirmation and closes settings only after successful persistence', async ({
  page,
}) => {
  await page.getByRole('button', { name: '설정', exact: true }).click();
  const modal = page.getByRole('dialog');
  await expect(modal.getByText('파일 탐색 열기', { exact: true })).toHaveCount(0);
  await expect(modal.getByText('스킬 탐색 열기', { exact: true })).toHaveCount(0);
  await modal.getByRole('button', { name: '실행', exact: true }).click();
  await modal.getByRole('button', { name: '저장', exact: true }).click();
  const confirm = page.getByRole('alertdialog');
  await expect(confirm).toContainText('저장하시겠습니까?');
  await confirm.getByRole('button', { name: '아니오', exact: true }).click();
  await expect(modal).toBeVisible();
  await modal.getByRole('button', { name: '저장', exact: true }).click();
  await confirm.getByRole('button', { name: '네', exact: true }).click();
  await expect(modal).toHaveCount(0);
  await expect(confirm).toHaveCount(0);
});

test('file previews and directory breadcrumbs wrap within narrow containers', async ({ page }) => {
  await page.getByRole('tab', { name: '파일', exact: true }).click();
  await page.getByRole('button', { name: 'polish-folder', exact: true }).click();
  const breadcrumb = page
    .locator('.directory-column')
    .getByText(/polish-folder$/, { exact: false })
    .first();
  expect(
    await breadcrumb.evaluate((node) => parseFloat(getComputedStyle(node).fontSize))
  ).toBeGreaterThanOrEqual(14);
  await page.getByRole('button', { name: 'wrap.md', exact: true }).click();
  await page.getByRole('radio', { name: 'Preview', exact: true }).click();
  await page.setViewportSize({ width: 480, height: 900 });
  await expect(page.locator('.file-document pre:visible')).toBeVisible();
  expect(await noOverflow(page, '.file-document')).toEqual([]);
  expect(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth)).toBe(false);
  await page.getByRole('radio', { name: 'Raw', exact: true }).click();
  const raw = page.getByRole('textbox', { name: /wrap\.md$/ });
  expect(await raw.evaluate((node) => node.scrollWidth <= node.clientWidth + 2)).toBe(true);
});

test('model save keeps the confirmation open on rejection and closes after a successful retry', async ({
  page,
}) => {
  await page.getByRole('button', { name: '설정', exact: true }).click();
  const modal = page.getByRole('dialog');
  await modal.getByRole('textbox', { name: /^Base URL/ }).fill(`${fixture.provider}/v1`);
  await modal.getByRole('combobox', { name: /^모델 ID/ }).click();
  await page.getByRole('option', { name: 'fixture-model', exact: true }).click();
  let reject = true;
  await page.route('**/v1/command', async (route) => {
    if (reject && route.request().postDataJSON()?.setModel) {
      reject = false;
      await route.fulfill({
        json: { error: { code: 'unavailable', message: 'Model settings fixture rejection' } },
      });
    } else await route.continue();
  });
  await modal.getByRole('button', { name: '저장', exact: true }).click();
  const confirm = page.getByRole('alertdialog');
  await confirm.getByRole('button', { name: '네', exact: true }).click();
  await expect(confirm).toContainText('Model settings fixture rejection');
  await expect(modal).toBeVisible();
  await confirm.getByRole('button', { name: '네', exact: true }).click();
  await expect(modal).toHaveCount(0);
  await expect(confirm).toHaveCount(0);
});

test('thinking and tool cells hold the same width through persistence and completion', async ({
  page,
}) => {
  try {
    await send(page, 'TOOL_CYCLE_FIXTURE');
    await expect(page.locator('.thinking')).toContainText('Stable initial tool reasoning');
    const width = (await page.locator('.thinking').first().boundingBox().then(required)).width;
    const contentWidth = (await page.locator('.chat-content').boundingBox().then(required)).width;
    const areaWidth = (await page.locator('#meca-work-area').boundingBox().then(required)).width;
    expect(contentWidth / areaWidth).toBeGreaterThan(0.75);
    expect(width / contentWidth).toBeGreaterThan(0.9);
    await page.evaluate(() => {
      const frames: boolean[] = [];
      let id = 0;
      const sample = () => {
        frames.push(
          [...document.querySelectorAll('.thinking')].some(
            (node) =>
              node.getClientRects().length > 0 &&
              node.textContent.includes('Stable initial tool reasoning')
          )
        );
        id = requestAnimationFrame(sample);
      };
      Object.assign(window, { cellFrames: frames, stopCellFrames: () => cancelAnimationFrame(id) });
      sample();
    });
    await page.request.get(`${fixture.provider}/test/start-tool`);
    await expect(page.locator('.thinking').last()).toContainText('Stable subsequent reasoning');
    await expect(page.locator('.tool-calls').first()).toBeVisible();
    expect(
      Math.abs(
        (await page.locator('.tool-entry').first().boundingBox().then(required)).width - width
      )
    ).toBeLessThan(2);
    expect(
      Math.abs((await page.locator('.thinking').first().boundingBox().then(required)).width - width)
    ).toBeLessThan(2);
    await page.setViewportSize({ width: 480, height: 900 });
    expect(await noOverflow(page, '.chat-content')).toEqual([]);
    await page.request.get(`${fixture.provider}/test/finish-tool`);
    await expect(page.locator('.completion-summary')).toBeVisible();
    await expect(page.locator('.completion-row')).not.toContainText('Intermediate response');
    await expect(page.locator('.completion-row').locator('[role=img]')).toHaveCount(0);
    expect(await noOverflow(page, '.chat-content')).toEqual([]);
    const order = await page.locator('.chat-content').evaluate((root) => {
      const summary = root.querySelector('.completion-summary');
      return [...root.querySelectorAll('.thinking, .tool-calls')].every(
        (node) => !!(node.compareDocumentPosition(summary!) & Node.DOCUMENT_POSITION_FOLLOWING)
      );
    });
    expect(order).toBe(true);
    const visible = await page.evaluate(() => {
      const state = window as unknown as { cellFrames: boolean[]; stopCellFrames: () => void };
      state.stopCellFrames();
      return state.cellFrames;
    });
    expect(visible.length).toBeGreaterThan(1);
    expect(visible.every(Boolean)).toBe(true);
  } finally {
    await page.request.get(`${fixture.provider}/test/start-tool`);
    await page.request.get(`${fixture.provider}/test/finish-tool`);
  }
});
