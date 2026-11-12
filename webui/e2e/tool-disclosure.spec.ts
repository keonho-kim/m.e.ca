// Purpose: Prevent async tool disclosures from overlapping responses or opening translucent result modals.
// Pattern: Isolated browser contracts with controlled native detail responses.
// Usage: Run against TestUIBrowserFixture through MECA_UI_DESCRIPTOR.
// Related: webui/src/pages/workspace/ui/chat/tool-call.tsx, webui/src/pages/workspace/ui/chat/chat.tsx.

import { expect, type Page } from '@playwright/test';
import {
  type BrowserFixture,
  browserFixture,
  connectBrowser,
  historyPage,
  required,
  test,
} from '#e2e/fixture';

let fixture: BrowserFixture;
test.beforeAll(async () => {
  fixture = await browserFixture();
});
async function openConversation(page: Page) {
  await connectBrowser(page, fixture);
  await page
    .locator('.workspace-group')
    .filter({ has: page.locator('.workspace-name', { hasText: 'alpha' }) })
    .getByRole('button', { name: '새 대화', exact: true })
    .click();
}
async function orderedRows(page: Page) {
  return page.locator('.history-row').evaluateAll((rows) =>
    rows.every((row, index) => {
      const next = rows[index + 1];
      return (
        !next ||
        (row as HTMLElement).offsetTop +
          Math.max((row as HTMLElement).offsetHeight, row.scrollHeight) <=
          (next as HTMLElement).offsetTop + 2
      );
    })
  );
}
const detailPage = (text: string) => ({
  detail: {
    text,
    totalBytes: String(Buffer.byteLength(text)),
    nextOffset: String(Buffer.byteLength(text)),
    revision: 'a'.repeat(64),
    contentType: 'application/json',
  },
});

test('async results stay inline, wrap and push later rows immediately in both themes', async ({
  page,
}) => {
  const stdout = 'unbroken_output_'.repeat(450);
  let release = () => {};
  const held = new Promise<void>((resolve) => {
    release = resolve;
  });
  const views: string[] = [];
  await page.route('**/v1/command', async (route) => {
    const command = route.request().postDataJSON();
    if (command.history) {
      await route.fulfill({
        json: {
          history: historyPage([
            { sequence: '1', message: { role: 'assistant', text: 'Before the tool.' } },
            {
              sequence: '2',
              event: {
                kind: 'tool_succeeded',
                agent: 'reviewer',
                tool: {
                  phase: 'TOOL_PHASE_SUCCEEDED',
                  name: 'execute_command',
                  reference: {
                    sessionId: command.history.sessionId,
                    sequence: '2',
                  },
                },
              },
            },
            {
              sequence: '3',
              event: {
                kind: 'thinking',
                message: { role: 'assistant', thinking: 'Reasoning after the tool.' },
              },
            },
            { sequence: '4', message: { role: 'assistant', text: 'Answer after the tool.' } },
          ]),
        },
      });
    } else if (command.detail) {
      views.push(command.detail.view);
      if (command.detail.view === 'TOOL_RESULT') await held;
      await route.fulfill({
        json: detailPage(
          JSON.stringify(
            command.detail.view === 'TOOL_REQUEST'
              ? { command: 'printf requested-command', path: '/workspace/example' }
              : {
                  stdout,
                  exit_code: 0,
                  command_id: 'execution-metadata',
                  ...Object.fromEntries(
                    Array.from({ length: 200 }, (_, index) => [`field_${index}`, `value_${index}`])
                  ),
                }
          )
        ),
      });
    } else await route.continue();
  });
  try {
    await openConversation(page);
    const tool = page.locator('.tool-entry');
    const trigger = tool.locator('.tool-disclosure > button');
    await expect(trigger).toHaveCount(1);
    const triggerBounds = await trigger.boundingBox().then(required);
    const requestBounds = await tool.locator('.tool-request-action').boundingBox().then(required);
    expect(requestBounds.x - triggerBounds.x - triggerBounds.width).toBeGreaterThanOrEqual(0);
    expect(requestBounds.x - triggerBounds.x - triggerBounds.width).toBeLessThanOrEqual(12);
    await expect(page.locator('.tool-message')).toHaveAccessibleName('reviewer');
    expect(views).toEqual([]);
    await trigger.focus();
    await page.keyboard.press('Tab');
    await page.keyboard.press('Shift+Tab');
    await expect(trigger).toBeFocused();
    const focus = await trigger.evaluate((node) => {
      const style = getComputedStyle(node);
      return {
        visible: node.matches(':focus-visible'),
        width: parseFloat(style.outlineWidth),
        style: style.outlineStyle,
        height: node.getBoundingClientRect().height,
      };
    });
    expect(focus.visible).toBe(true);
    expect(focus.width).toBeGreaterThan(0);
    expect(focus.style).not.toBe('none');
    expect(focus.height).toBeGreaterThanOrEqual(32);
    await trigger.press('Enter');
    await expect(tool.locator('.detail-content')).toBeVisible();
    expect(await orderedRows(page)).toBe(true);
    release();
    await expect(tool.locator('.meca-code')).toContainText(stdout);
    expect(await orderedRows(page)).toBe(true);
    await expect(tool.getByText('execution-metadata', { exact: true })).toHaveCount(0);
    await tool.getByRole('button', { name: '실행 정보', exact: true }).click();
    await expect(tool).toContainText('execution-metadata');
    expect(await orderedRows(page)).toBe(true);
    await expect(page.getByRole('dialog')).toHaveCount(0);
    for (const theme of ['light', 'dark']) {
      if (theme === 'dark')
        await page.getByRole('button', { name: '화면 테마', exact: true }).click();
      await trigger.hover();
      const chrome = await tool.locator('.tool-calls').evaluate((node) => {
        const style = getComputedStyle(node);
        return { background: style.backgroundColor, decoration: style.textDecorationLine };
      });
      expect(chrome.background).toBe('rgba(0, 0, 0, 0)');
      expect(chrome.decoration).toContain('underline');
      await tool.getByRole('button', { name: /요청 내용/ }).click();
      const dialog = page.getByRole('dialog');
      await expect(dialog).toContainText('printf requested-command');
      await expect(dialog).not.toContainText(stdout);
      const title = await dialog.getByRole('heading').boundingBox().then(required);
      const label = await dialog.getByText('명령', { exact: true }).boundingBox().then(required);
      expect(Math.abs(title.x - label.x)).toBeLessThan(2);
      const subtitle = await dialog
        .getByText('요청 내용', { exact: true })
        .boundingBox()
        .then(required);
      expect(label.y - subtitle.y - subtitle.height).toBeGreaterThanOrEqual(16);
      const material = await dialog.evaluate((node) => {
        const style = getComputedStyle(node);
        return {
          background: style.backgroundColor,
          filter: style.backdropFilter,
          opacity: style.opacity,
          animation: style.animationName,
        };
      });
      expect(material.background).toMatch(/^rgb\(/);
      expect(material.filter).toBe('none');
      expect(material.opacity).toBe('1');
      expect(['none', 'meca-dialog-enter']).toContain(material.animation);
      await page.keyboard.press('Escape');
    }
    for (const width of [1440, 768, 390]) {
      await page.setViewportSize({ width, height: 900 });
      expect(await orderedRows(page)).toBe(true);
      expect(await tool.evaluate((node) => node.scrollWidth <= node.clientWidth + 2)).toBe(true);
      expect(
        await tool
          .locator('.meca-code')
          .evaluateAll((nodes) => nodes.every((node) => node.scrollWidth <= node.clientWidth + 2))
      ).toBe(true);
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(
        true
      );
    }
    expect(views.filter((view) => view === 'TOOL_RESULT')).toHaveLength(1);
    expect(views.filter((view) => view === 'TOOL_REQUEST')).toHaveLength(2);
    await expect(page.getByRole('button', { name: /View complete text|원문 보기/ })).toHaveCount(0);
  } finally {
    release();
  }
});

test('long saved narratives page inline and virtual history keeps a bounded mounted window', async ({
  page,
}) => {
  const prefix = `Stored narrative page one.\n\n${'paragraph words '.repeat(1500)}`;
  const suffix = 'Stored narrative page two ends here.';
  const offset = String(Buffer.byteLength(prefix));
  let expire = true;
  await page.route('**/v1/command', async (route) => {
    const command = route.request().postDataJSON();
    if (command.history)
      await route.fulfill({
        json: {
          history: historyPage(
            Array.from({ length: 32 }, (_, index) => ({
              sequence: String(index + 1),
              truncated: index === 31,
              message: {
                role: 'assistant',
                text:
                  index === 31
                    ? 'Stored narrative excerpt.'
                    : `History ${index}. ${'history words '.repeat(100)}`,
              },
            }))
          ),
        },
      });
    else if (command.detail) {
      expect(command.detail.view).toBe('TEXT');
      if (command.detail.offset !== '0' && expire) {
        expire = false;
        await route.fulfill({
          json: { error: { code: 'conflict', message: 'Detail revision changed.' } },
        });
        return;
      }
      await route.fulfill({
        json: {
          detail: {
            text: command.detail.offset === '0' ? prefix : suffix,
            totalBytes: String(Buffer.byteLength(prefix + suffix)),
            nextOffset:
              command.detail.offset === '0' ? offset : String(Buffer.byteLength(prefix + suffix)),
            revision: 'b'.repeat(64),
            contentType: 'text/plain',
          },
        },
      });
    } else await route.continue();
  });
  await openConversation(page);
  const scroller = page.locator('.workspace-body');
  await expect(page.locator('.history-row').first()).toBeVisible();
  await scroller.evaluate((node) => {
    node.scrollTop = 0;
  });
  await expect(page.getByText(/^History 0\./)).toBeVisible();
  expect(await page.locator('.history-row').count()).toBeLessThan(24);
  await scroller.evaluate((node) => {
    node.scrollTop = node.scrollHeight;
  });
  const detail = page.locator('.detail-content');
  await expect(detail).toContainText('Stored narrative page one.');
  await detail.getByRole('button', { name: '다음 페이지', exact: true }).click();
  await detail.getByRole('button', { name: '다시 확인', exact: true }).click();
  await expect(detail).toContainText('Stored narrative page one.');
  await detail.getByRole('button', { name: '다음 페이지', exact: true }).click();
  await expect(detail).toContainText(suffix);
  await detail.getByRole('button', { name: '이전 페이지', exact: true }).click();
  await expect(detail).toContainText('Stored narrative page one.');
  expect(await orderedRows(page)).toBe(true);
  expect(await page.locator('.history-row').count()).toBeLessThan(24);
  await expect(page.getByRole('dialog')).toHaveCount(0);
});
