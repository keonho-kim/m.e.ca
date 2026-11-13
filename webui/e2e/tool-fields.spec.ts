// Purpose: Keep oversized structured tool output readable without exposing partial JSON fragments.
// Pattern: Browser contract over independently revised native field pages.
// Usage: MECA_UI_DESCRIPTOR points to TestUIBrowserFixture.
// Related: webui/src/pages/workspace/ui/chat/structured-tool.tsx, internal/serverapi/detail/fields.go.
import { expect } from '@playwright/test';
import { browserFixture, connectBrowser, create, historyPage, test } from '#e2e/fixture';

test('large tool fields page independently and release their disclosures on collapse', async ({
  page,
}) => {
  await page.route('**/v1/command', async (route) => {
    const command = route.request().postDataJSON();
    if (command.history) {
      await route.fulfill({
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
    } else if (command.detail) {
      const source = command.detail;
      const second = source.offset !== '0';
      if (second) expect(source.revision).toBe('c'.repeat(64));
      const text = source.listFields
        ? '["stdout","stderr","exit_code"]'
        : source.field === 'stdout'
          ? second
            ? 'Second stdout page.'
            : 'First stdout page.'
          : source.field === 'stderr'
            ? 'No stderr.'
            : source.field
              ? '0'
              : '{"stdout":"incomplete raw fragment';
      const more = !source.listFields && (!source.field || (source.field === 'stdout' && !second));
      await route.fulfill({
        json: {
          detail: {
            text,
            totalBytes: more ? '90000' : String(second ? 90000 : text.length),
            nextOffset: more ? '32768' : String(second ? 90000 : text.length),
            revision: (source.field ? 'c' : 'a').repeat(64),
            contentType: source.field ? 'text/plain' : 'application/json',
          },
        },
      });
    } else await route.continue();
  });
  await connectBrowser(page, await browserFixture());
  await create(page);
  const trigger = page.locator('.tool-disclosure > button').first();
  await trigger.click();
  await expect(page.getByText('First stdout page.', { exact: true })).toBeVisible();
  await expect(page.getByText('No stderr.', { exact: true })).toBeVisible();
  await expect(page.getByText('incomplete raw fragment', { exact: false })).toHaveCount(0);
  await page.getByRole('button', { name: '다음 페이지', exact: true }).click();
  await expect(page.getByText('Second stdout page.', { exact: true })).toBeVisible();
  await trigger.click();
  await expect(page.locator('.detail-content')).toHaveCount(0);
});
