// Purpose: Keep local native selection, cancellation and unavailable-desktop fallback distinct.
// Pattern: Real workspace registration with a controlled desktop HTTP response.
// Usage: MECA_UI_DESCRIPTOR points to the isolated Go fixture.
// Related: webui/src/pages/workspace/model/workspace-picker.ts, internal/serverapi/ui_folder_test.go.
import { mkdir } from 'node:fs/promises';
import { join } from 'node:path';
import { expect } from '@playwright/test';
import { browserFixture, connectBrowser, test } from '#e2e/fixture';

test('local native cancellation leaves the current workspace unchanged and selection registers its canonical identity', async ({
  page,
}) => {
  const fixture = await browserFixture();
  await connectBrowser(page, fixture);
  await page.unroute('**/v1/ui/pick-folder');
  let path = '';
  await page.route('**/v1/ui/pick-folder', (route) => route.fulfill({ json: { path } }));
  const before = page.url();
  const add = page.getByRole('button', { name: '워크스페이스 추가', exact: true });
  const canceled = page.waitForResponse((response) =>
    response.url().endsWith('/v1/ui/pick-folder')
  );
  await add.click();
  await canceled;
  await expect(add).toBeEnabled();
  await expect(page).toHaveURL(before);
  await expect(page.getByRole('dialog')).toHaveCount(0);
  path = join(fixture.home, 'native-selected');
  await mkdir(path, { recursive: true });
  await add.click();
  await expect(
    page.locator('.workspace-name').filter({ hasText: 'native-selected' })
  ).toBeVisible();
  await expect(page).not.toHaveURL(before);
  await expect(page.getByRole('dialog')).toHaveCount(0);
});

test('native failure, timeout and busy states keep a usable folder fallback and a safe diagnosis', async ({
  page,
}) => {
  const fixture = await browserFixture();
  await connectBrowser(page, fixture);
  await page.unroute('**/v1/ui/pick-folder');
  let status = 500;
  await page.route('**/v1/ui/pick-folder', (route) =>
    route.fulfill({ status, body: 'private OS diagnostic' })
  );
  for (const [code, message] of [
    [500, 'OS 폴더 선택창을 열지 못했습니다.'],
    [504, 'OS 폴더 선택 시간이 초과되었습니다.'],
    [409, 'OS 폴더 선택창이 이미 열려 있습니다.'],
  ] as const) {
    status = code;
    await page.getByRole('button', { name: '워크스페이스 추가', exact: true }).click();
    const dialog = page.getByRole('dialog', { name: '워크스페이스 추가', exact: true });
    await expect(dialog).toBeVisible();
    await expect(dialog.getByRole('button', { name: '폴더 선택', exact: true })).toBeVisible();
    await dialog.getByRole('button', { name: '닫기', exact: true }).last().click();
    await expect(page.getByText(message, { exact: false })).toBeVisible();
    await expect(page.getByText('private OS diagnostic', { exact: false })).toHaveCount(0);
  }
});
