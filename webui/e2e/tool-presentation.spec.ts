// Purpose: Verify one row per real command and distinct reasoning, response and completion surfaces.
// Pattern: Browser-to-engine presentation regression.
// Usage: Run with the isolated TestUIBrowserFixture descriptor.
// Related: cmd/meca/ui_tool_provider_test.go, webui/src/pages/workspace/ui/chat/live-activity.tsx.
import { readFile } from 'node:fs/promises';
import { expect, test } from '@playwright/test';

test('repeated commands never duplicate messages or become completion metadata', async ({
  page,
}) => {
  test.skip(!process.env.MECA_UI_DESCRIPTOR, 'start the isolated browser fixture first');
  const fixture = JSON.parse(await readFile(process.env.MECA_UI_DESCRIPTOR!, 'utf8'));
  await page.goto(`${fixture.url}/ui/`);
  await page
    .getByRole('textbox', { name: '서버 토큰', exact: true })
    .fill((await readFile(fixture.tokenFile, 'utf8')).trim());
  await page.getByRole('button', { name: '연결', exact: true }).click();
  const previous = page.url();
  await page
    .locator('.workspace-group')
    .filter({ has: page.locator('.workspace-name', { hasText: 'alpha' }) })
    .getByRole('button', { name: '새 대화', exact: true })
    .click();
  await page.waitForURL((url) => url.href !== previous && url.pathname.includes('/c/session_'));
  await page
    .getByRole('combobox', { name: '작업 입력', exact: true })
    .fill('REPEATED_TOOL_FIXTURE');
  await page.locator('.composer').getByRole('button', { name: '전송', exact: true }).click();
  await expect(page.getByText('Before first command.', { exact: true })).toBeVisible();
  await page.request.get(fixture.provider + '/test/tool-sequence/1');
  await expect(page.getByText('Before second command.', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: /execute_command/ })).toHaveCount(1);
  await expect(page.getByText('Before first command.', { exact: true })).toHaveCount(1);
  await page.reload();
  await expect(page.getByText('Before second command.', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: /execute_command/ })).toHaveCount(1);
  await page.request.get(fixture.provider + '/test/tool-sequence/2');
  await expect(page.getByText('Final response outside statistics.', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: /execute_command/ })).toHaveCount(2);
  await expect(page.getByText('Before second command.', { exact: true })).toHaveCount(1);
  await page.request.get(fixture.provider + '/test/tool-sequence/3');
  await expect(page.locator('.completion-summary')).toContainText('입력 32');
  await expect(page.locator('.completion-row')).not.toContainText('command.');
  await expect(page.locator('.completion-row')).not.toContainText('Final response');
  for (const text of [
    'Before first command.',
    'Before second command.',
    'Final response outside statistics.',
  ]) {
    await expect(page.getByText(text, { exact: true })).toHaveCount(1);
  }
  const answer = page.getByText('Final response outside statistics.', { exact: true });
  const thought = page.locator('.thinking').last();
  const gap = await answer.evaluate((node) => {
    const thinking = node.closest('article')!.querySelector('.thinking')!;
    return node.getBoundingClientRect().top - thinking.getBoundingClientRect().bottom;
  });
  expect(gap).toBeGreaterThanOrEqual(12);
  expect(await thought.evaluate((node) => getComputedStyle(node).backgroundColor)).not.toBe(
    'rgba(0, 0, 0, 0)'
  );
  expect(
    await thought.locator('.meca-prose').evaluate((node) => getComputedStyle(node).color)
  ).not.toBe(await answer.evaluate((node) => getComputedStyle(node).color));
});
