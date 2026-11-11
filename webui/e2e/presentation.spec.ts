// Purpose: Verify full names, scaled settings and accurate unsaved-model instructions.
// Pattern: Rendered presentation regression contracts.
// Usage: Run with MECA_UI_DESCRIPTOR against the isolated embedded UI fixture.
// Related: webui/src/pages/workspace/ui/settings/settings.tsx, cmd/meca/ui_browser_test.go.
import { mkdir, readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { expect, test } from '@playwright/test';

let fixture: { url: string; tokenFile: string; home: string };
test.beforeAll(async () => {
  const descriptor = process.env.MECA_UI_DESCRIPTOR;
  test.skip(!descriptor, 'start the isolated UI fixture first');
  if (descriptor) fixture = JSON.parse(await readFile(descriptor, 'utf8'));
});
test.beforeEach(async ({ page }) => {
  await page.goto(`${fixture.url}/ui/`);
  await page
    .getByRole('textbox', { name: '서버 토큰', exact: true })
    .fill((await readFile(fixture.tokenFile, 'utf8')).trim());
  await page.getByRole('button', { name: '연결', exact: true }).click();
  const previous = page.url();
  const group = page
    .locator('.workspace-group')
    .filter({ has: page.locator('.workspace-name', { hasText: 'alpha' }) });
  await group.getByRole('button', { name: '새 대화', exact: true }).click();
  await page.waitForURL((url) => url.href !== previous && url.pathname.includes('/c/session_'));
});
test('closing a dirty model draft describes model settings rather than file changes', async ({
  page,
}) => {
  await page.getByRole('button', { name: '설정', exact: true }).click();
  const settings = page.getByRole('dialog', { name: '설정', exact: true });
  const address = settings.getByRole('textbox', { name: 'Base URL', exact: true });
  await address.fill((await address.inputValue()) + '/draft');
  await settings.getByRole('button', { name: '닫기', exact: true }).click();
  await expect(
    page.getByText('모델 설정 변경을 저장하거나 버린 뒤 닫으세요.', { exact: true })
  ).toBeVisible();
  await page.getByRole('button', { name: '계속 편집', exact: true }).click();
  await expect(address).toHaveValue(/\/draft$/);
});
test('settings names remain visible at double text size and compact theme choices do not truncate', async ({
  page,
}) => {
  await page.getByRole('button', { name: '설정', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: '설정', exact: true });
  await page.evaluate(() => {
    document.documentElement.style.fontSize = '200%';
  });
  for (const label of ['모델', '실행', '일반']) {
    const button = dialog.getByRole('button', { name: label, exact: true });
    await expect(button).toBeVisible();
    expect(
      await button.evaluate((element) =>
        [...element.querySelectorAll('*')].some((node) => {
          if (!(node instanceof HTMLElement)) return false;
          const style = getComputedStyle(node);
          return style.textOverflow === 'ellipsis' && node.scrollWidth > node.clientWidth;
        })
      )
    ).toBe(false);
  }
  await page.evaluate(() => {
    document.documentElement.style.fontSize = '';
  });
  await dialog.getByRole('button', { name: '일반', exact: true }).click();
  await page.setViewportSize({ width: 320, height: 844 });
  await dialog.getByRole('combobox', { name: '화면 테마', exact: true }).click();
  for (const label of ['라이트', '다크', '시스템']) {
    const option = page.getByRole('option', { name: label, exact: true });
    await expect(option).toBeVisible();
    expect(await option.evaluate((element) => element.scrollWidth <= element.clientWidth + 1)).toBe(
      true
    );
  }
});
test('a long project name exposes its full path through the native tooltip', async ({ page }) => {
  const name = '고객_데이터_분석_프로젝트_운영_워크스페이스_장기보관';
  const path = join(fixture.home, name);
  await mkdir(path, { recursive: true });
  await page.getByRole('button', { name: '워크스페이스 추가', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: '워크스페이스 추가', exact: true });
  await dialog.getByRole('textbox', { name: '경로', exact: true }).fill(path);
  await dialog.getByRole('textbox', { name: '경로', exact: true }).press('Enter');
  await expect(dialog.getByRole('textbox', { name: '경로', exact: true })).toHaveAttribute(
    'placeholder',
    new RegExp(name)
  );
  await dialog.getByRole('button', { name: '폴더 선택', exact: true }).click();
  const item = page.locator('.workspace-name').filter({ hasText: name });
  await expect(item).toBeVisible();
  await item.hover();
  await expect(page.getByRole('tooltip')).toContainText(path);
});
