// Purpose: Verify guarded file writes, skill mentions, global model setup and task approvals.
// Pattern: Browser-to-Go behavior contracts with isolated model fixtures.
// Usage: MECA_UI_DESCRIPTOR points to the opt-in Go browser fixture.
// Related: webui/e2e/fixture.ts, cmd/meca/ui_browser_test.go.

import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { expect } from '@playwright/test';
import {
  type BrowserFixture,
  browserFixture,
  connectBrowser,
  create,
  input,
  required,
  rpc,
  send,
  test,
} from '#e2e/fixture';

let fixture: BrowserFixture;
test.beforeAll(async () => {
  fixture = await browserFixture();
});
test.beforeEach(async ({ page }) => {
  await connectBrowser(page, fixture);
});
test('files use complete revision-bound saves and skills insert canonical mentions', async ({
  page,
}) => {
  await writeFile(
    join(fixture.projectA, 'sample.go'),
    'package sample\n\nfunc Value() int { return 1 }\n'
  );
  await create(page);
  await send(page, '!dir');
  await page.getByRole('button', { name: '파일 작업 · sample.go', exact: true }).click();
  await page.getByRole('menuitem', { name: '추가하기', exact: true }).click();
  await page.getByRole('button', { name: 'sample.go', exact: true }).click();
  await page
    .getByRole('textbox', { name: /sample\.go$/ })
    .fill('package sample\n\nfunc Value() int { return 42 }\n');
  await page.getByRole('button', { name: '저장', exact: true }).click();
  await expect(page.getByRole('button', { name: '저장', exact: true })).toBeDisabled();
  await expect
    .poll(() => readFile(join(fixture.projectA, 'sample.go'), 'utf8'))
    .toContain('return 42');
  await page
    .getByRole('textbox', { name: /sample\.go$/ })
    .fill('package sample\n\nfunc Value() int { return 43 }\n');
  await writeFile(
    join(fixture.projectA, 'sample.go'),
    'package sample\n\nfunc Value() int { return 99 }\n'
  );
  await page.getByRole('button', { name: '저장', exact: true }).click();
  await expect(page.getByText('file changed', { exact: false })).toBeVisible();
  await expect(page.getByRole('textbox', { name: /sample\.go$/ })).toContainText('return 43');
  await page
    .locator('.document-toolbar')
    .getByRole('button', { name: '취소', exact: true })
    .click();
  await page.getByRole('button', { name: '변경 버리기', exact: true }).click();
  await page.locator('.workspace-header').getByRole('tab', { name: '스킬', exact: true }).click();
  await page.locator('.skill-entry').filter({ hasText: 'ui-design' }).click();
  await expect(page.locator('.skill-document')).toContainText('ui-design');
  await page
    .locator('.skill-entry')
    .filter({ hasText: 'ui-design' })
    .getByRole('button', { name: /입력에 스킬 넣기/ })
    .click();
  await expect(input(page)).toHaveText(/#meca:ui-design/);
  await send(page, '!model');
  const modal = page.getByRole('dialog');
  await expect(modal.getByRole('combobox', { name: 'Provider', exact: true })).toContainText(
    'vLLM'
  );
  await modal.getByRole('button', { name: '닫기', exact: true }).click();
  await page.setViewportSize({ width: 390, height: 844 });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
});

test('folder registration, authenticated images and task-bound approval are operational', async ({
  page,
}) => {
  const added = join(fixture.home, 'gamma');
  await mkdir(added, { recursive: true });
  await page.getByRole('button', { name: '워크스페이스 추가', exact: true }).click();
  let modal = page.getByRole('dialog', { name: '워크스페이스 추가', exact: true });
  await modal.getByRole('textbox', { name: '경로', exact: true }).fill(added);
  await modal.getByRole('textbox', { name: '경로', exact: true }).press('Enter');
  await expect(modal.getByText(/\/gamma$/).first()).toBeVisible();
  await modal.getByRole('button', { name: '폴더 선택', exact: true }).click();
  await expect(page.locator('.workspace-name').filter({ hasText: 'gamma' })).toBeVisible();
  await create(page);
  const choosing = page.waitForEvent('filechooser');
  await page.getByRole('button', { name: '첨부 및 스킬', exact: true }).click();
  await page.getByRole('menuitem', { name: '이미지 첨부', exact: true }).click();
  await (await choosing).setFiles(join(fixture.home, 'fixture.png'));
  await send(page, 'Inspect the attached fixture');
  const image = page.getByRole('group', { name: '이미지 첨부', exact: true }).first();
  await expect(image).toBeVisible();
  await image.getByRole('button').click();
  modal = page.getByRole('dialog');
  await expect
    .poll(() => modal.locator('img').evaluate((image: HTMLImageElement) => image.naturalWidth))
    .toBe(16);
  await modal.getByRole('button', { name: '닫기', exact: true }).click();
  await create(page);
  await send(page, 'APPROVAL_FIXTURE');
  modal = page.getByRole('dialog', { name: '승인 요청', exact: true });
  await expect(modal).toBeVisible();
  await modal.getByRole('button', { name: '승인', exact: true }).click();
  await expect(modal).toBeHidden();
  await expect(page.locator('.history-rows')).toContainText('Completed: APPROVAL_FIXTURE');
  expect(await readFile(join(fixture.home, 'approval-written.txt'), 'utf8')).toBe(
    'approved fixture\n'
  );
});

test('mobile settings discover, diagnose and save the global model before opening a conversation', async ({
  page,
}) => {
  await expect(page).toHaveURL(/\/w\/[^/]+$/);
  await page.setViewportSize({ width: 390, height: 844 });
  await page.getByRole('button', { name: '열기', exact: true }).click();
  await page.getByRole('button', { name: '설정', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: '설정', exact: true });
  await expect(dialog.getByRole('combobox', { name: '설정 항목', exact: true })).toBeVisible();
  await expect
    .poll(
      async () =>
        (
          await dialog
            .getByRole('textbox', { name: /^Base URL/ })
            .boundingBox()
            .then(required)
        ).width
    )
    .toBeGreaterThan(280);
  await expect(dialog.getByRole('button', { name: '모델 조회', exact: true })).toHaveCount(0);
  await dialog.getByRole('textbox', { name: /^Base URL/ }).fill(`${fixture.provider}/`);
  const catalog = page.waitForResponse(
    (response) =>
      response.url().endsWith('/v1/command') && !!response.request().postDataJSON()?.discoverModels
  );
  await dialog.getByRole('combobox', { name: /^모델 ID/ }).click();
  expect((await (await catalog).json()).error).toBeUndefined();
  await page.getByRole('option', { name: 'fixture-model', exact: true }).click();
  for (const [label, key] of [
    ['연결 진단', 'diagnoseModel'],
    ['저장', 'setModel'],
  ] as const) {
    const reply = page.waitForResponse(
      (response) =>
        response.url().endsWith('/v1/command') &&
        response.request().postDataJSON()?.[key] !== undefined
    );
    await dialog.getByRole('button', { name: label, exact: true }).click();
    if (key === 'setModel')
      await page.getByRole('alertdialog').getByRole('button', { name: '네', exact: true }).click();
    const response = await reply;
    expect(response.request().postDataJSON()[key].sessionId).toBeUndefined();
    expect((await response.json()).error).toBeUndefined();
  }
  expect((await rpc(page, { state: { defaultModel: true } })).state?.model?.model).toBe(
    'fixture-model'
  );
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
});
