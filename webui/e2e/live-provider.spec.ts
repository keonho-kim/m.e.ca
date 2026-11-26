// Purpose: Verify model discovery, saving and one reply against an explicitly selected LM Studio server.
// Pattern: On-demand browser integration with an external model endpoint.
// Usage: Set MECA_LMSTUDIO_URL and MECA_LMSTUDIO_MODEL, then run bun run test:live-provider.
// Related: webui/playwright.probes.config.ts, webui/e2e/fixture.ts.
import { expect } from '@playwright/test';
import { connectBrowser, create, rpc, send, test } from '#e2e/fixture';

test('a discovered LM Studio model can be saved and answer a request', async ({ page, server }) => {
  const endpoint = process.env.MECA_LMSTUDIO_URL;
  const model = process.env.MECA_LMSTUDIO_MODEL;
  if (!endpoint || !model) throw new Error('Set MECA_LMSTUDIO_URL and MECA_LMSTUDIO_MODEL');
  test.setTimeout(120000);
  await connectBrowser(page, server);
  const conversation = new URL(await create(page));
  await page.getByRole('button', { name: '설정', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: '설정', exact: true });
  await dialog.getByRole('combobox', { name: 'Provider', exact: true }).click();
  await page.getByRole('option', { name: 'LM Studio', exact: true }).click();
  await dialog.getByRole('textbox', { name: /^Base URL/ }).fill(endpoint);
  await dialog.getByRole('combobox', { name: /^모델 ID/ }).click();
  const candidate = page.getByRole('option', { name: model, exact: true });
  await candidate.click({ timeout: 15000 });
  await dialog.getByRole('button', { name: '저장', exact: true }).click();
  await page.getByRole('alertdialog').getByRole('button', { name: '네', exact: true }).click();
  await expect(dialog).toBeHidden({ timeout: 30000 });
  const started = page.waitForResponse(
    (response) =>
      response.url().endsWith('/v1/command') && !!response.request().postDataJSON()?.start
  );
  await send(page, 'Reply with only OK. Do not use tools.');
  const task = (await (await started).json()).task;
  expect(task?.id).toBeTruthy();
  const workspace = conversation.pathname.split('/')[3];
  await expect
    .poll(
      async () => (await rpc(page, { taskStatus: { taskId: task.id } }, workspace)).task?.status,
      {
        timeout: 90000,
      }
    )
    .toBe('answered');
  await expect(page.locator('.history-rows')).toContainText('OK');
});
