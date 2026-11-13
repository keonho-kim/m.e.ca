// Purpose: Exercise first-login, folder creation, attachment menus and real local model discovery.
// Pattern: Browser-to-engine usability regression contracts.
// Usage: Run with MECA_UI_DESCRIPTOR and optional MECA_LMSTUDIO_URL against the isolated UI fixture.
// Related: cmd/meca/ui_browser_test.go, webui/src/pages/workspace/ui/navigation/folder-picker.tsx.

import { readFile } from 'node:fs/promises';
import { expect, type Page } from '@playwright/test';
import { type BrowserFixture, browserFixture, connectBrowser, required, test } from '#e2e/fixture';

let fixture: BrowserFixture;
test.beforeAll(async () => {
  fixture = await browserFixture();
});
async function login(page: Page, entireLine = false) {
  await page.goto(`${fixture.url}/ui/`);
  const token = (await readFile(fixture.tokenFile, 'utf8')).trim();
  await page
    .getByRole('textbox', { name: /^서버 토큰/ })
    .fill(entireLine ? `Server token: ${token}` : token);
  await page.getByRole('button', { name: '연결', exact: true }).click();
  await expect(page.getByRole('navigation', { name: '워크스페이스', exact: true })).toBeVisible();
}
async function createChat(page: Page) {
  const group = page
    .locator('.workspace-group')
    .filter({ has: page.locator('.workspace-name', { hasText: 'alpha' }) });
  await group.getByRole('button', { name: '새 대화', exact: true }).click();
  await expect(page.getByRole('combobox', { name: '작업 입력', exact: true })).toBeEnabled();
}

test('an anonymous first visit does not call protected APIs and the printed token line authenticates', async ({
  page,
}) => {
  const rejected: string[] = [];
  page.on('response', (response) => {
    if (response.status() === 401) rejected.push(new URL(response.url()).pathname);
  });
  await login(page, true);
  expect(rejected).toEqual([]);
  await page.reload();
  await expect(page.getByRole('navigation', { name: '워크스페이스', exact: true })).toBeVisible();
  expect(rejected).toEqual([]);
});

test('sidebar controls, settings labels and skill insertion stay in their work area', async ({
  page,
}) => {
  await connectBrowser(page, fixture);
  await createChat(page);
  const collapse = page.getByRole('button', { name: '사이드바 접기', exact: true });
  expect((await collapse.boundingBox().then(required)).y).toBeLessThan(80);
  const settings = page.getByRole('button', { name: '설정', exact: true });
  expect((await settings.boundingBox().then(required)).y).toBeGreaterThan(
    required(page.viewportSize()).height - 100
  );
  await collapse.click();
  await page
    .getByRole('navigation', { name: '워크스페이스', exact: true })
    .getByRole('button', { name: '사이드바 펼치기', exact: true })
    .click();
  await settings.click();
  const dialog = page.getByRole('dialog', { name: '설정', exact: true });
  expect((await dialog.boundingBox().then(required)).width).toBeGreaterThan(950);
  for (const label of ['모델', '실행']) {
    const item = dialog.getByRole('button', { name: label, exact: true });
    await expect(item).toBeVisible();
    expect(
      await item.evaluate((element) =>
        [...element.querySelectorAll('*')].some(
          (node) =>
            node instanceof HTMLElement &&
            getComputedStyle(node).textOverflow === 'ellipsis' &&
            node.scrollWidth > node.clientWidth
        )
      )
    ).toBe(false);
  }
  await dialog.getByRole('button', { name: '닫기', exact: true }).click();
  const composer = page.locator('.composer');
  await composer.getByRole('button', { name: '팀 모드', exact: true }).hover();
  await expect(page.getByRole('tooltip')).toContainText('팀 모드');
  await composer.getByRole('button', { name: '첨부 및 스킬', exact: true }).click();
  await page.getByRole('menuitem', { name: '스킬 목록 보기', exact: true }).click();
  const skills = page.getByRole('dialog', { name: '스킬', exact: true });
  await expect(skills).toBeVisible();
  await skills.getByRole('button', { name: /create-skill.*입력에 스킬 넣기/ }).click();
  await expect(skills).toBeHidden();
  await expect(page.getByRole('tab', { name: '대화', exact: true })).toHaveAttribute(
    'aria-selected',
    'true'
  );
  await expect(page.getByRole('combobox', { name: '작업 입력', exact: true })).toContainText('#');
  await expect(page.getByRole('combobox', { name: '작업 입력', exact: true })).toBeFocused();
  const choosing = page.waitForEvent('filechooser');
  await composer.getByRole('button', { name: '첨부 및 스킬', exact: true }).click();
  await page.getByRole('menuitem', { name: '이미지 첨부', exact: true }).click();
  await (await choosing).setFiles(`${fixture.home}/fixture.png`);
  await expect(page.getByRole('group', { name: /^fixture\.png/ })).toBeVisible();
});

test('workspace folder creation and explicit selection work without a clipped open button', async ({
  page,
}) => {
  const name = `workspace-${crypto.randomUUID()}`;
  await connectBrowser(page, fixture);
  await page.getByRole('button', { name: '워크스페이스 추가', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: '워크스페이스 추가', exact: true });
  expect((await dialog.boundingBox().then(required)).width).toBeGreaterThan(850);
  await dialog.getByRole('textbox', { name: '경로', exact: true }).fill(fixture.home);
  await dialog.getByRole('textbox', { name: '경로', exact: true }).press('Enter');
  await dialog.getByRole('button', { name: '새 폴더', exact: true }).click();
  await dialog.getByRole('textbox', { name: '폴더 이름', exact: true }).fill(name);
  await dialog.getByRole('button', { name: '만들기', exact: true }).click();
  await expect(dialog.getByRole('textbox', { name: '폴더 이름', exact: true })).toBeHidden();
  await dialog.getByRole('button', { name: '폴더 선택', exact: true }).click();
  await expect(dialog).toBeHidden();
  await expect(page.locator('.workspace-name').filter({ hasText: name })).toBeVisible();
  await page.getByRole('button', { name: '워크스페이스 추가', exact: true }).click();
  await dialog.getByRole('textbox', { name: '경로', exact: true }).fill(fixture.home);
  await dialog.getByRole('textbox', { name: '경로', exact: true }).press('Enter');
  await dialog.getByRole('button', { name: 'beta', exact: true }).dblclick();
  await expect(dialog).toBeHidden();
});

test('global model candidates follow the server address before a conversation is selected', async ({
  page,
}) => {
  await connectBrowser(page, fixture);
  await page.route('**/v1/command', async (route) => {
    const command = route.request().postDataJSON();
    if (!command?.discoverModels) return route.continue();
    const id = command.discoverModels.endpoint.endsWith(':10001')
      ? 'catalog-first'
      : 'catalog-second';
    await route.fulfill({ json: { models: { items: [{ id }] } } });
  });
  await page.getByRole('button', { name: '설정', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: '설정', exact: true });
  const address = dialog.getByRole('textbox', { name: /^Base URL/ });
  const model = dialog.getByRole('combobox', { name: /^모델 ID/ });
  await expect(address).toBeVisible();
  await expect(dialog.getByRole('button', { name: '모델 조회', exact: true })).toHaveCount(0);
  await address.fill('http://127.0.0.1:10001');
  await model.click();
  await expect(page.getByRole('option', { name: 'catalog-first', exact: true })).toBeVisible();
  await address.fill('http://127.0.0.1:10002');
  await model.click();
  await expect(page.getByRole('option', { name: 'catalog-second', exact: true })).toBeVisible();
  await expect(page.getByRole('option', { name: 'catalog-first', exact: true })).toBeHidden();
});

test('LM Studio supplies Ornith candidates on model focus and accepts the selected model', async ({
  page,
}) => {
  test.skip(
    !process.env.MECA_LMSTUDIO_URL,
    'requires the explicitly requested local LM Studio model'
  );
  test.setTimeout(120000);
  await connectBrowser(page, fixture);
  try {
    await createChat(page);
    await page.getByRole('button', { name: '설정', exact: true }).click();
    const dialog = page.getByRole('dialog', { name: '설정', exact: true });
    await dialog.getByRole('combobox', { name: 'Provider', exact: true }).click();
    await page.getByRole('option', { name: 'LM Studio', exact: true }).click();
    await dialog.getByRole('textbox', { name: /^Base URL/ }).fill(process.env.MECA_LMSTUDIO_URL!);
    const model = dialog.getByRole('combobox', { name: /^모델 ID/ });
    await model.click();
    await expect(page.getByRole('option', { name: 'ornith-1.5-35b-a3b', exact: true })).toBeVisible(
      {
        timeout: 15000,
      }
    );
    await page.getByRole('option', { name: 'ornith-1.5-35b-a3b', exact: true }).click();
    await dialog.getByRole('button', { name: '저장', exact: true }).click();
    await page.getByRole('alertdialog').getByRole('button', { name: '네', exact: true }).click();
    await expect(dialog).toHaveCount(0, { timeout: 30000 });
    const started = page.waitForResponse(
      (response) =>
        response.url().endsWith('/v1/command') && !!response.request().postDataJSON()?.start
    );
    await page
      .getByRole('combobox', { name: '작업 입력', exact: true })
      .fill(
        '도구를 사용하지 마세요. 영문 대문자 O 다음에 영문 대문자 K를 붙여서 두 글자로만 답하세요.'
      );
    await page.getByRole('button', { name: /^(전송|보내기)$/, exact: true }).click();
    const task = (await (await started).json()).task;
    expect(task?.id).toBeTruthy();
    await expect
      .poll(
        async () =>
          page.evaluate(async (id) => {
            const response = await fetch('/v1/command', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json', 'X-Meca-UI': '1' },
              body: JSON.stringify({ taskStatus: { taskId: id } }),
            });
            return (await response.json()).task?.status;
          }, task.id),
        { timeout: 90000 }
      )
      .toBe('answered');
    await expect(page.locator('.history-rows')).toContainText('OK');
  } finally {
    const restored = await page.request.post(`${fixture.url}/v1/command`, {
      headers: { Origin: fixture.url, 'X-Meca-UI': '1' },
      data: {
        setModel: {
          defaultModel: true,
          provider: 'vllm',
          endpoint: fixture.provider,
          model: 'fixture-model',
        },
      },
    });
    expect((await restored.json()).error).toBeUndefined();
  }
});
