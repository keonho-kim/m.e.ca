// Purpose: Verify real parallel runtime calls render under one agent heading after persistence and reload.
// Pattern: Browser-to-engine grouping and completion contracts.
// Usage: MECA_UI_DESCRIPTOR points to the deterministic Go browser fixture.
// Related: cmd/meca/ui_parallel_provider_test.go, webui/src/pages/workspace/ui/chat/tool-call.tsx.
import { expect } from '@playwright/test';
import { browserFixture, connectBrowser, create, send, test } from '#e2e/fixture';

test('one parallel wave has one agent heading, independent tool disclosures and typed final metrics', async ({
  page,
}) => {
  const fixture = await browserFixture();
  await connectBrowser(page, fixture);
  await create(page);
  await send(page, 'PARALLEL_FIXTURE');
  await expect(page.getByText('Parallel fixtures inspected.', { exact: true })).toBeVisible();
  const tools = page.locator('.tool-message');
  await expect(tools).toHaveCount(1);
  await expect(tools.locator('.tool-entry')).toHaveCount(2);
  await expect(tools.getByText('MECA', { exact: true })).toHaveCount(1);
  const summary = page.locator('.completion-summary');
  await expect(summary).toContainText('입력 64');
  await expect(summary).toContainText('출력 128');
  await expect(summary).not.toContainText('완료');
  await expect(summary).not.toContainText('Parallel fixtures inspected.');
  await page.reload();
  await expect(page.locator('.tool-message')).toHaveCount(1);
  await expect(page.locator('.tool-message .tool-entry')).toHaveCount(2);
});
