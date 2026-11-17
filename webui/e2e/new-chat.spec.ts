// Purpose: Verify one empty conversation surface and opaque settings selections.
// Pattern: Browser behavior regression against the isolated server.
// Usage: Run with the opt-in MECA_UI_DESCRIPTOR fixture.
// Related: webui/src/pages/workspace/ui/workspace-page.tsx, webui/src/app/styles/theme.ts.
import { expect } from '@playwright/test';
import { browserFixture, connectBrowser, create, input, send, test } from '#e2e/fixture';

test('workspace selection enables the same empty chat without a new-chat click', async ({
  page,
}) => {
  await page.route('**/v1/command', async (route) => {
    if (!route.request().postDataJSON().listWorkspaces) return route.continue();
    const response = await route.fetch();
    const reply = await response.json();
    delete reply.workspaces.defaultId;
    await route.fulfill({ response, json: reply });
  });
  await connectBrowser(page, await browserFixture());
  await expect(input(page)).toHaveAttribute('contenteditable', 'false');
  await expect(page.getByRole('heading', { name: '어떤 작업을 시작할까요?' })).toBeVisible();
  await expect(page.getByRole('button', { name: '대화 검색', exact: true })).toHaveCount(0);
  await page.locator('.workspace-name').filter({ hasText: 'alpha' }).click();
  await expect(input(page)).toHaveAttribute('contenteditable', 'true');
  await expect(page.getByRole('heading', { name: '어떤 작업을 시작할까요?' })).toBeVisible();
  const landing = await page.locator('.composer').boundingBox();
  await page.locator('input[type=file]').setInputFiles({
    name: 'first-image.png',
    mimeType: 'image/png',
    buffer: Buffer.from(
      'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR4nGMQMgn7DwAClAGcHVtGXwAAAABJRU5ErkJggg==',
      'base64'
    ),
  });
  await expect(page.getByRole('region', { name: '첨부 이미지', exact: true })).toBeVisible();
  await send(page, 'First request without creating a conversation');
  await expect(page).toHaveURL(/\/c\/session_/);
  await expect(input(page)).toHaveText('');
  await expect(page.getByRole('region', { name: '첨부 이미지', exact: true })).toHaveCount(0);
  await expect(page.getByRole('button', { name: '대화 검색', exact: true })).toBeVisible();
  await create(page);
  await expect(page.getByRole('heading', { name: '어떤 작업을 시작할까요?' })).toBeVisible();
  await expect(page.getByRole('button', { name: '대화 검색', exact: true })).toHaveCount(0);
  await expect
    .poll(async () => {
      const created = await page.locator('.composer').boundingBox();
      return Math.abs((created?.y ?? -1000) - (landing?.y ?? 0));
    })
    .toBeLessThan(2);
});

for (const theme of ['light', 'dark']) {
  test(`settings selections are opaque in ${theme}`, async ({ page }) => {
    await connectBrowser(page, await browserFixture());
    if (theme === 'dark')
      await page.getByRole('button', { name: '화면 테마', exact: true }).click();
    await page.getByRole('button', { name: '설정', exact: true }).click();
    const dialog = page.getByRole('dialog', { name: '설정', exact: true });
    const provider = dialog.getByRole('combobox', { name: 'Provider', exact: true });
    await provider.click();
    const surface = page.locator('.astryx-popover:visible').last();
    const opaque = async () => {
      await expect(surface).toBeVisible();
      const fill = await surface.evaluate((node) => {
        const canvas = document.createElement('canvas');
        canvas.width = canvas.height = 1;
        const ctx = canvas.getContext('2d');
        if (!ctx) throw Error('Canvas unavailable');
        ctx.fillStyle = getComputedStyle(node).backgroundColor;
        ctx.fillRect(0, 0, 1, 1);
        return ctx.getImageData(0, 0, 1, 1).data[3];
      });
      expect(fill).toBe(255);
      expect(await surface.evaluate((node) => getComputedStyle(node).backdropFilter)).toBe('none');
    };
    await opaque();
    await page.keyboard.press('Escape');
    await expect(provider).toBeFocused();
    await dialog.getByRole('button', { name: '선택 해제', exact: true }).click();
    await dialog.getByRole('combobox', { name: /^모델 ID/ }).click();
    await opaque();
    await page.screenshot({ path: `../.local/new-chat-check/settings-${theme}.png` });
  });
}

test('a rejected creation keeps the draft and retry submits it once', async ({ page }) => {
  await connectBrowser(page, await browserFixture());
  await page.locator('.workspace-name').filter({ hasText: 'alpha' }).click();
  let creations = 0;
  let starts = 0;
  await page.route('**/v1/command', async (route) => {
    const command = route.request().postDataJSON();
    if (command.start) starts++;
    if (command.selectSession?.create && ++creations === 1)
      return route.fulfill({ status: 503, json: { error: { code: 'unavailable' } } });
    await route.continue();
  });
  await send(page, 'Preserve this draft');
  await expect(input(page)).toHaveAttribute('contenteditable', 'true');
  await expect(input(page)).toHaveText('Preserve this draft');
  expect(starts).toBe(0);
  await page.getByRole('button', { name: /^(전송|보내기)$/ }).click();
  await expect(page).toHaveURL(/\/c\/session_/);
  await expect(input(page)).toHaveText('');
  expect(creations).toBe(2);
  expect(starts).toBe(1);
});

test('creation does not redirect or submit after switching workspaces', async ({ page }) => {
  await connectBrowser(page, await browserFixture());
  await page.locator('.workspace-name').filter({ hasText: 'alpha' }).click();
  let release = () => {};
  const held = new Promise<void>((resolve) => {
    release = resolve;
  });
  let arrived = () => {};
  const ready = new Promise<void>((resolve) => {
    arrived = resolve;
  });
  let finished = () => {};
  const settled = new Promise<void>((resolve) => {
    finished = resolve;
  });
  let starts = 0;
  await page.route('**/v1/command', async (route) => {
    const command = route.request().postDataJSON();
    if (command.start) starts++;
    if (!command.selectSession?.create) return route.continue();
    const response = await route.fetch();
    arrived();
    await held;
    await route.fulfill({ response });
    finished();
  });
  try {
    await send(page, 'Keep alpha draft after navigation');
    await ready;
    await page.locator('.workspace-name').filter({ hasText: 'beta' }).click();
    const destination = page.url();
    release();
    await settled;
    await expect(input(page)).toHaveAttribute('contenteditable', 'true');
    await expect(page).toHaveURL(destination);
    await page.locator('.workspace-name').filter({ hasText: 'alpha' }).click();
    await expect(input(page)).toHaveText('Keep alpha draft after navigation');
    expect(starts).toBe(0);
  } finally {
    release();
  }
});

test('team command and toggle prepare a conversation and preserve draft ownership', async ({
  page,
}) => {
  await connectBrowser(page, await browserFixture());
  await page.locator('.workspace-name').filter({ hasText: 'alpha' }).click();
  await send(page, '!team');
  await expect(page).toHaveURL(/\/c\/session_/);
  await expect(page.getByRole('button', { name: '팀 모드', exact: true })).toHaveAttribute(
    'aria-pressed',
    'true'
  );
  await expect(input(page)).toHaveText('');
  await page.locator('.workspace-name').filter({ hasText: 'beta' }).click();
  await expect(page).toHaveURL(/\/w\/[^/]+$/);
  await expect(page.getByRole('button', { name: '팀 모드', exact: true })).toHaveAttribute(
    'aria-pressed',
    'false'
  );
  await input(page).fill('Team draft');
  await page.getByRole('button', { name: '팀 모드', exact: true }).click();
  await expect(page).toHaveURL(/\/c\/session_/);
  await expect(page.getByRole('button', { name: '팀 모드', exact: true })).toHaveAttribute(
    'aria-pressed',
    'true'
  );
  await expect(input(page)).toHaveText('Team draft');
  await expect(page.getByRole('button', { name: '대화 검색', exact: true })).toHaveCount(0);
});

test('unsaved files prevent session creation without moving the draft', async ({ page }) => {
  await connectBrowser(page, await browserFixture());
  await page.locator('.workspace-name').filter({ hasText: 'alpha' }).click();
  await input(page).fill('Draft beside an edited file');
  await page.getByRole('tab', { name: '파일', exact: true }).click();
  await page.getByRole('button', { name: 'sample.go', exact: true }).click();
  await page.getByRole('textbox', { name: /sample\.go$/ }).fill('package changed');
  let creations = 0;
  await page.route('**/v1/command', async (route) => {
    if (route.request().postDataJSON().selectSession?.create) creations++;
    await route.continue();
  });
  await page.getByRole('button', { name: '팀 모드', exact: true }).click();
  await expect(
    page.getByText('작성 중인 내용을 저장하거나 버린 뒤 이동해 주세요.', { exact: true })
  ).toBeVisible();
  expect(creations).toBe(0);
  await expect(input(page)).toHaveText('Draft beside an edited file');
  await expect(page.getByRole('textbox', { name: /sample\.go$/ })).toHaveValue('package changed');
});
