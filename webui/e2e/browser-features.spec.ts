// Purpose: Exercise bounded image previews, worker serialization and server-backed conversation search.
// Pattern: Native browser feature contracts over an isolated project and model fixture.
// Usage: Run through the Playwright browser suite.
// Related: webui/src/shared/lib/image-preview/index.ts, internal/serverapi/history_search.go.
import { expect } from '@playwright/test';
import { connectBrowser, create, input, rpc, send, test } from '#e2e/fixture';

test('search reaches saved messages outside the mounted transcript', async ({
  page,
  server: fixture,
}) => {
  await connectBrowser(page, fixture);
  const { workspaces } = await rpc(page, { listWorkspaces: {} });
  const workspace = workspaces?.defaultId;
  expect(workspace).toBeTruthy();
  await page.goto(`${fixture.url}/ui/w/${workspace}/c/${fixture.archiveSession}`);
  await expect(page.getByText('Archive message 299', { exact: true })).toBeVisible();
  const target = 'Archive message 250';
  await expect(page.getByText(target, { exact: true })).toHaveCount(0);
  await page.getByRole('button', { name: '대화 검색', exact: true }).click();
  const search = page.getByRole('dialog', { name: '대화 검색', exact: true });
  await search.getByRole('searchbox', { name: /^대화 검색/ }).fill(target);
  await search.getByRole('button', { name: '검색', exact: true }).click();
  const match = search.getByRole('button', { name: target, exact: true });
  const more = search.getByRole('button', { name: '다음 구간 검색', exact: true });
  for (let pageIndex = 0; pageIndex < 4 && !(await match.isVisible()); pageIndex++) {
    await expect(more).toBeEnabled();
    const searched = page.waitForResponse(
      (response) =>
        response.url().endsWith('/v1/command') &&
        !!response.request().postDataJSON()?.searchHistory?.cursor
    );
    await more.click();
    await (await searched).finished();
  }
  await expect(match).toBeVisible();
  await match.click();
  await expect(search).toBeHidden();
  await expect(page.getByText(target, { exact: true })).toBeVisible();
  await expect(
    page.getByRole('button', { name: '최근 대화로 돌아가기', exact: true })
  ).toBeVisible();
});

test('large command serialization uses a short-lived worker', async ({ page, server: fixture }) => {
  await connectBrowser(page, fixture);
  await create(page);
  const worker = page.waitForEvent('worker');
  await send(page, 'fixture '.repeat(35000));
  const active = await worker;
  expect(active.url()).toContain('json-encoder');
  await expect(input(page)).toHaveText('');
  await expect.poll(() => page.workers().length).toBe(0);
});

test('plain-text typing uses native edit ranges and preserves a large middle edit', async ({
  server: fixture,
  page,
}) => {
  await connectBrowser(page, fixture);
  await create(page);
  await input(page).fill('x'.repeat(300000));
  await input(page).evaluate((node) => {
    const original = InputEvent.prototype.getTargetRanges;
    Object.assign(window, { inputRangeReads: 0 });
    InputEvent.prototype.getTargetRanges = function () {
      Reflect.set(window, 'inputRangeReads', Number(Reflect.get(window, 'inputRangeReads')) + 1);
      return original.call(this);
    };
    if (!node.firstChild) throw Error('Plain input missing');
    const range = document.createRange();
    range.setStart(node.firstChild, 150000);
    range.collapse(true);
    const selection = getSelection();
    selection?.removeAllRanges();
    selection?.addRange(range);
  });
  await page.keyboard.press('a');
  expect(await input(page).evaluate((node) => node.textContent.slice(149999, 150002))).toBe('xax');
  expect(await page.evaluate(() => Reflect.get(window, 'inputRangeReads'))).toBeGreaterThan(0);
});

test('preview images use bounded derived pixels and release their URL when removed', async ({
  server: fixture,
  page,
}) => {
  await page.addInitScript(() => {
    const original = URL.revokeObjectURL.bind(URL);
    const values: string[] = [];
    Object.assign(window, { releasedPreviews: values });
    URL.revokeObjectURL = (url) => {
      values.push(url);
      original(url);
    };
  });
  await connectBrowser(page, fixture);
  await create(page);
  const bytes = await page.evaluate(async () => {
    const canvas = document.createElement('canvas');
    canvas.width = 2048;
    canvas.height = 1024;
    const context = canvas.getContext('2d');
    if (!context) throw Error('Canvas unavailable');
    context.fillStyle = '#123456';
    context.fillRect(0, 0, canvas.width, canvas.height);
    const blob = await new Promise<Blob>((resolve) =>
      canvas.toBlob((value) => {
        if (value) resolve(value);
      })
    );
    return Array.from(new Uint8Array(await blob.arrayBuffer()));
  });
  await page.locator('input[type=file]').setInputFiles({
    name: 'preview-fixture.png',
    mimeType: 'image/png',
    buffer: Buffer.from(bytes),
  });
  const image = page.locator('.composer-image img');
  await expect(image).toHaveCount(1);
  await expect
    .poll(() => image.evaluate((node) => (node as HTMLImageElement).naturalWidth))
    .toBe(256);
  const url = await image.getAttribute('src');
  await page.locator('.composer-image').hover();
  await page.getByRole('button', { name: /preview-fixture.png.*제거/ }).click();
  await expect(image).toHaveCount(0);
  expect(
    await page.evaluate(
      (url) =>
        (window as unknown as { releasedPreviews: string[] }).releasedPreviews.includes(url ?? ''),
      url
    )
  ).toBe(true);
});
