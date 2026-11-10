// Purpose: Verify usable controls, document modes and layout boundaries with realistic content.
// Pattern: Browser-to-engine regression contracts over the isolated UI fixture.
// Usage: Run with MECA_UI_DESCRIPTOR after rebuilding the embedded UI.
// Related: cmd/meca/ui_browser_test.go, webui/src/pages/workspace/ui/files/file-document.tsx.
import { readFile, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { expect, type Locator, type Page, test } from '@playwright/test';

let fixture: { url: string; tokenFile: string; projectA: string };
const file = 'layout-review.md';
const original =
  '# Layout review\n\n```sql\nSELECT ' +
  'customer_revenue_column, '.repeat(60) +
  'id FROM reports;\n```\n';
test.beforeAll(async () => {
  const descriptor = process.env.MECA_UI_DESCRIPTOR;
  test.skip(!descriptor, 'start the isolated UI fixture first');
  if (!descriptor) return;
  fixture = JSON.parse(await readFile(descriptor, 'utf8'));
  await writeFile(join(fixture.projectA, file), original);
});
async function create(page: Page) {
  const previous = page.url();
  const group = page
    .locator('.workspace-group')
    .filter({ has: page.locator('.workspace-name', { hasText: 'alpha' }) });
  await group.getByRole('button', { name: '새 대화', exact: true }).click();
  await page.waitForURL((url) => url.href !== previous && url.pathname.includes('/c/session_'));
  await expect(page.getByRole('combobox', { name: '작업 입력', exact: true })).toBeEnabled();
}
test.beforeEach(async ({ page }) => {
  await page.goto(`${fixture.url}/ui/`);
  await page
    .getByRole('textbox', { name: '서버 토큰', exact: true })
    .fill((await readFile(fixture.tokenFile, 'utf8')).trim());
  await page.getByRole('button', { name: '연결', exact: true }).click();
  await expect(page.getByRole('navigation', { name: '워크스페이스', exact: true })).toBeVisible();
  await create(page);
});
async function openFile(page: Page) {
  await page.getByRole('tab', { name: '파일', exact: true }).click();
  await page.getByRole('button', { name: file, exact: true }).click();
  await expect(page.getByRole('textbox', { name: /layout-review\.md$/ })).toBeVisible();
}
async function hitTarget(control: Locator) {
  return control.evaluate((element) => {
    const bounds = element.getBoundingClientRect();
    const hit = document.elementFromPoint(
      bounds.x + bounds.width / 2,
      bounds.y + bounds.height / 2
    );
    return !!hit && (element === hit || element.contains(hit));
  });
}
test('the empty composer accepts pointer typing after settings closes', async ({ page }) => {
  const input = page.getByRole('combobox', { name: '작업 입력', exact: true });
  await page.getByRole('button', { name: '설정', exact: true }).click();
  const settings = page.getByRole('dialog', { name: '설정', exact: true });
  await expect(settings.getByRole('combobox', { name: 'Provider', exact: true })).toBeVisible();
  await settings.getByRole('button', { name: '닫기', exact: true }).click();
  await expect.poll(async () => (await input.boundingBox())?.height ?? 0).toBeGreaterThan(0);
  await input.click();
  await input.pressSequentially('Pointer typing works');
  await expect(input).toHaveText('Pointer typing works');
});
test('work area tabs occupy a row above the project and conversation title', async ({ page }) => {
  const tabs = await page.getByRole('tablist', { name: '작업 화면', exact: true }).boundingBox();
  const context = await page
    .locator('.workspace-header')
    .getByText('alpha', { exact: true })
    .boundingBox();
  expect(tabs!.y + tabs!.height).toBeLessThanOrEqual(context!.y);
});
test('long code stays inside its document and leaves file controls reachable', async ({ page }) => {
  await openFile(page);
  const preview = page.getByRole('radio', { name: 'Preview', exact: true });
  expect(await hitTarget(preview)).toBe(true);
  const bounds = await preview.boundingBox();
  expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(page.viewportSize()!.width);
  await expect(page.getByRole('radio', { name: 'Raw', exact: true })).toBeChecked();
  await preview.click();
  await expect(preview).toBeChecked();
  await page.getByRole('radio', { name: 'Raw', exact: true }).click();
  await expect(page.getByRole('radio', { name: 'Raw', exact: true })).toBeChecked();
  expect(await hitTarget(preview)).toBe(true);
  await page.setViewportSize({ width: 320, height: 844 });
  await expect.poll(async () => {
    const selector = await page.getByRole('radiogroup', { name: '파일 보기', exact: true }).boundingBox();
    const panel = await page.locator('.file-document').boundingBox();
    return panel!.x + panel!.width - selector!.x - selector!.width;
  }).toBeLessThan(20);
  expect(await hitTarget(preview)).toBe(true);
});
test('Markdown preview reads unsaved edits and preserves editor undo and redo', async ({
  page,
}) => {
  await openFile(page);
  const editor = page.getByRole('textbox', { name: new RegExp(file.replace('.', '\\.')) });
  await editor.fill('# Unsaved preview\n\n**Draft content**');
  await editor.pressSequentially('!');
  const text = () =>
    editor.evaluate((element) =>
      element instanceof HTMLTextAreaElement ? element.value : element.textContent
    );
  const edited = await text();
  await page.getByRole('radio', { name: 'Preview', exact: true }).click();
  await expect(
    page.locator('.file-document').getByRole('heading', { name: 'Unsaved preview', exact: true })
  ).toBeVisible();
  await expect(page.locator('.file-document strong')).toHaveText('Draft content');
  expect(await readFile(join(fixture.projectA, file), 'utf8')).toBe(original);
  await page.getByRole('radio', { name: 'Raw', exact: true }).click();
  expect(await text()).toBe(edited);
  const modifier = process.platform === 'darwin' ? 'Meta' : 'Control';
  await editor.focus();
  await editor.press(`${modifier}+z`);
  await expect.poll(text).not.toBe(edited);
  await editor.press(`${modifier}+Shift+z`);
  await expect.poll(text).toBe(edited);
  await page.getByRole('button', { name: '저장', exact: true }).click();
  await expect.poll(() => readFile(join(fixture.projectA, file), 'utf8')).toBe(edited);
  await writeFile(join(fixture.projectA, file), original);
});
test('a narrow skill dialog shows a readable document and can return to its list', async ({
  page,
}) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.getByRole('button', { name: '첨부 및 스킬', exact: true }).click();
  await page.getByRole('menuitem', { name: '스킬 목록 보기', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: '스킬', exact: true });
  const frame = await dialog.boundingBox();
  const title = await dialog.getByRole('heading', { name: '스킬', exact: true }).boundingBox();
  expect(title!.x - frame!.x).toBeGreaterThan(8);
  expect(title!.y - frame!.y).toBeGreaterThan(8);
  await dialog
    .locator('.skill-entry')
    .filter({ hasText: 'ui-design' })
    .getByRole('button')
    .first()
    .click();
  const document = dialog.locator('.skill-document');
  await expect(document.getByRole('heading', { name: 'UI DESIGN', exact: true })).toBeVisible();
  expect((await document.boundingBox())!.width).toBeGreaterThan(280);
  await dialog.getByRole('button', { name: '스킬 목록으로', exact: true }).click();
  await expect(dialog.getByRole('textbox', { name: '스킬 검색', exact: true })).toBeVisible();
});
test('long queued requests leave edit and remove actions visible to a pointer', async ({
  page,
}) => {
  const input = page.getByRole('combobox', { name: '작업 입력', exact: true });
  await input.fill('HOLD_FIXTURE layout queue');
  await page.getByRole('button', { name: /^(전송|보내기)$/, exact: true }).click();
  const stop = page.getByRole('button', { name: '작업 중지', exact: true });
  await expect(stop).toBeVisible();
  try {
    await input.fill('followup_without_breaks_'.repeat(30));
    await input.press('Control+Enter');
    await page.setViewportSize({ width: 390, height: 844 });
    const queue = page.getByRole('region', { name: '예약 요청', exact: true });
    for (const label of ['편집', '제거']) {
      const action = queue.getByRole('button', { name: label, exact: true });
      await expect(action).toBeVisible();
      expect(await hitTarget(action)).toBe(true);
    }
    await input.fill('Steer the active task');
    await expect(page.getByRole('button', { name: '방향 수정', exact: true })).toBeEnabled();
    await page.getByRole('button', { name: '작업', exact: true }).click();
    const drawer = page.getByRole('dialog', { name: '작업', exact: true });
    const running = drawer.getByRole('listitem').filter({ hasText: '실행 중' });
    await expect(
      running.getByRole('button', { name: 'HOLD_FIXTURE layout queue', exact: true })
    ).toBeVisible();
    await expect(running.getByText('실행 중', { exact: true })).toBeVisible();
    await drawer.getByRole('button', { name: '닫기', exact: true }).click();
  } finally {
    const drawer = page.getByRole('dialog', { name: '작업', exact: true });
    if (await drawer.count())
      await drawer.getByRole('button', { name: '닫기', exact: true }).click();
    if (await stop.count()) {
      await stop.click();
      await expect(stop).toBeHidden();
    }
  }
});
