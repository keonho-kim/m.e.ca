// Purpose: Keep large tool requests lazy, bounded, copyable and inside their owning disclosure.
// Pattern: Browser contract with revision-bound synthetic request pages.
// Usage: MECA_UI_DESCRIPTOR points to TestUIBrowserFixture.
// Related: webui/src/pages/workspace/ui/chat/tool-request.tsx, webui/src/pages/workspace/model/detail-pages.ts.
import { expect } from '@playwright/test';
import { browserFixture, connectBrowser, create, historyPage, test } from '#e2e/fixture';

test('request pages share the result width and release on collapse', async ({ page }) => {
  const first = `{"content":"${'request content '.repeat(1900)}first-page-end`,
    second = 'last-request-page"}',
    revision = 'c'.repeat(64);
  const reads: { offset: string; revision?: string }[] = [];
  await page.addInitScript(() => {
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText: async (text: string) => Object.assign(window, { copiedRequest: text }) },
    });
  });
  await page.route('**/v1/command', async (route) => {
    const command = route.request().postDataJSON();
    if (command.history)
      return route.fulfill({
        json: {
          history: historyPage([
            {
              sequence: '1',
              event: {
                kind: 'tool_succeeded',
                tool: {
                  name: 'execute_command',
                  phase: 'TOOL_PHASE_SUCCEEDED',
                  reference: { sessionId: command.history.sessionId, sequence: '1' },
                },
              },
            },
          ]),
        },
      });
    if (command.detail) {
      const request = command.detail.view === 'TOOL_REQUEST';
      if (request) {
        expect(command.detail.limit).toBe(32768);
        reads.push({ offset: command.detail.offset, revision: command.detail.revision });
      }
      const text = request
        ? command.detail.offset === '0'
          ? first
          : second
        : '{"stdout":"Visible result"}';
      return route.fulfill({
        json: {
          detail: {
            text,
            revision,
            totalBytes: String(request ? first.length + second.length : text.length),
            nextOffset: String(
              request && command.detail.offset === '0'
                ? first.length
                : request
                  ? first.length + second.length
                  : text.length
            ),
            contentType: 'application/json',
          },
        },
      });
    }
    return route.continue();
  });
  await connectBrowser(page, await browserFixture());
  await create(page);
  const tool = page.locator('.tool-entry');
  const outer = tool.locator('.tool-disclosure > button');
  await outer.click();
  await expect(tool).toContainText('Visible result');
  expect(reads).toEqual([]);
  const toggle = tool.getByRole('button', { name: '요청 원문', exact: true });
  await expect(toggle).toHaveAttribute('aria-expanded', 'false');
  await toggle.click();
  const viewport = tool.locator('.tool-request-content');
  await expect(viewport).toContainText('request content');
  for (const width of [1440, 768, 390]) {
    await page.setViewportSize({ width, height: 900 });
    const geometry = await viewport.evaluate((node) => {
      const owner = node.closest('.tool-entry');
      if (!owner) throw Error('Tool entry is missing');
      const row = owner.getBoundingClientRect(),
        box = node.getBoundingClientRect();
      return {
        width: box.width,
        expectedWidth: row.width,
        x: box.x,
        expectedX: row.x,
        height: box.height,
        scroll: node.scrollHeight,
        client: node.clientHeight,
        overflow: node.scrollWidth > node.clientWidth + 1,
      };
    });
    expect(Math.abs(geometry.width - geometry.expectedWidth)).toBeLessThanOrEqual(1);
    expect(Math.abs(geometry.x - geometry.expectedX)).toBeLessThanOrEqual(1);
    expect(geometry.height).toBeLessThanOrEqual(360);
    expect(geometry.scroll).toBeGreaterThan(geometry.client);
    expect(geometry.overflow).toBe(false);
  }
  await expect(page.getByRole('dialog')).toHaveCount(0);
  await viewport
    .getByRole('group', { name: '원문 범위', exact: true })
    .getByRole('button', { name: '다음 페이지', exact: true })
    .click();
  await expect(viewport).toContainText(second);
  expect(reads).toEqual([{ offset: '0' }, { offset: String(first.length), revision }]);
  await viewport.locator('.meca-code').hover();
  await viewport.getByRole('button', { name: /복사|Copy/ }).click();
  expect(
    await page.evaluate(() => (window as unknown as { copiedRequest: string }).copiedRequest)
  ).toBe(second);
  await outer.click();
  await expect(viewport).toHaveCount(0);
  await outer.click();
  await expect(toggle).toHaveAttribute('aria-expanded', 'false');
  await toggle.click();
  await expect(viewport).toContainText('request content');
  expect(reads.at(-1)?.offset).toBe('0');
  expect(reads).toHaveLength(3);
});
