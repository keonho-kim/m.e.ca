// Purpose: Verify real parallel runtime calls render under one agent heading after persistence and reload.
// Pattern: Browser-to-engine grouping and completion contracts.
// Usage: Run through the Playwright browser suite.
// Related: cmd/meca/ui_parallel_provider_test.go, webui/src/pages/workspace/ui/chat/tool-call.tsx.
import { expect } from '@playwright/test';
import { connectBrowser, create, send, test } from '#e2e/fixture';

test('one parallel wave has one agent heading, independent tool disclosures and typed final metrics', async ({
  server: fixture,
  page,
}) => {
  await connectBrowser(page, fixture);
  await create(page);
  await send(page, 'PARALLEL_FIXTURE');
  await expect(page.getByText('Parallel fixtures inspected.', { exact: true })).toBeVisible();
  const tools = page.locator('.tool-message');
  await expect(tools).toHaveCount(1);
  await expect(tools.locator('.tool-entry')).toHaveCount(3);
  await expect(tools.getByText('codemode', { exact: true })).toHaveCount(1);
  await expect(tools.getByText('execute_command', { exact: true })).toHaveCount(2);
  const code = tools.locator('.tool-entry').filter({ hasText: 'codemode' });
  await code.getByRole('button', { name: '완료 codemode', exact: true }).click();
  await expect(code.locator('.tool-content')).toContainText('completed');
  await code.getByRole('button', { name: '요청 원문', exact: true }).click();
  await expect(code.locator('.tool-request-content')).toContainText('Promise.all');
  await expect(tools.getByText('MECA', { exact: true })).toHaveCount(1);
  const summary = page.locator('.completion-summary');
  await expect(summary).toContainText('입력 64');
  await expect(summary).toContainText('출력 128');
  await expect(summary).not.toContainText('완료');
  await expect(summary).not.toContainText('Parallel fixtures inspected.');
  await page.reload();
  await expect(page.locator('.tool-message')).toHaveCount(1);
  await expect(page.locator('.tool-message .tool-entry')).toHaveCount(3);
});
