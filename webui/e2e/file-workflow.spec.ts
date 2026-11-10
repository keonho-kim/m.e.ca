// Purpose: Verify one right-side file list, direct editing and wrapped Markdown code blocks.
// Pattern: Browser-to-engine file workflow contracts.
// Usage: Run with MECA_UI_DESCRIPTOR against the isolated embedded UI fixture.
// Related: cmd/meca/ui_browser_test.go, webui/src/pages/workspace/ui/files/files.tsx.
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { expect, test } from '@playwright/test';

let fixture: { url: string; tokenFile: string; projectA: string };
const filename = 'inline-edit.md';
const source =
  '# Click to edit\n\nCurrent document.\n\n```text\n' +
  'unbrokenIdentifier_'.repeat(100) +
  '\n```\n';
test.beforeAll(async () => {
  const descriptor = process.env.MECA_UI_DESCRIPTOR;
  test.skip(!descriptor, 'start the isolated UI fixture first');
  if (!descriptor) return;
  fixture = JSON.parse(await readFile(descriptor, 'utf8'));
  await writeFile(join(fixture.projectA, filename), source);
  await mkdir(join(fixture.projectA, 'files-one', 'nested'), { recursive: true });
  await writeFile(join(fixture.projectA, 'files-one', 'nested', 'inside.txt'), 'Nested file');
  await writeFile(join(fixture.projectA, 'large-readonly.txt'), 'bounded text\n'.repeat(100000));
});
test.beforeEach(async ({ page }) => {
  await page.goto(`${fixture.url}/ui/`);
  await page
    .getByRole('textbox', { name: '서버 토큰', exact: true })
    .fill((await readFile(fixture.tokenFile, 'utf8')).trim());
  await page.getByRole('button', { name: '연결', exact: true }).click();
  await expect(page.getByRole('navigation', { name: '워크스페이스', exact: true })).toBeVisible();
  const previous = page.url();
  await page
    .locator('.workspace-group')
    .filter({ has: page.locator('.workspace-name', { hasText: 'alpha' }) })
    .getByRole('button', { name: '새 대화', exact: true })
    .click();
  await page.waitForURL((url) => url.href !== previous && url.pathname.includes('/c/session_'));
  await page.getByRole('tab', { name: '파일', exact: true }).click();
});
test('folder navigation keeps one file list to the right of the document', async ({ page }) => {
  await page.getByRole('button', { name: 'files-one', exact: true }).click();
  await page.getByRole('button', { name: 'nested', exact: true }).click();
  await expect(page.locator('.directory-column')).toHaveCount(1);
  await page.getByRole('button', { name: 'inside.txt', exact: true }).click();
  await expect(page.getByRole('textbox', { name: /inside\.txt$/ })).toBeVisible();
  const files = await page.locator('.directory-column').boundingBox();
  const document = await page.locator('.file-document').boundingBox();
  expect(files!.x).toBeGreaterThanOrEqual(document!.x + document!.width - 1);
  await expect(page.getByRole('navigation', { name: '워크스페이스', exact: true })).toBeVisible();
  await expect(
    page.locator('.files-workspace').getByRole('toolbar', { name: '파일', exact: true })
  ).toHaveCount(0);
  const listing = page.locator('.directory-column');
  const refresh = listing.getByRole('button', { name: '새로고침', exact: true });
  await expect(listing.getByText('작업공간', { exact: true })).toBeVisible();
  await expect(refresh).toBeVisible();
  const bounds = await refresh.boundingBox();
  expect(files!.x + files!.width - bounds!.x - bounds!.width).toBeLessThan(20);
  const refreshed = page.waitForResponse(
    (response) =>
      response.url().endsWith('/v1/command') && !!response.request().postDataJSON()?.directory
  );
  await refresh.click();
  await refreshed;
  await expect(
    page.locator('.file-document').getByRole('button', { name: '상위 폴더', exact: true })
  ).toHaveCount(0);
  await listing.getByRole('button', { name: '상위 폴더', exact: true }).click();
  await expect(listing.getByRole('button', { name: 'nested', exact: true })).toBeVisible();
});
test('selecting a file edits directly and clicking Markdown preview restores focused Raw editing', async ({
  page,
}) => {
  await page.getByRole('button', { name: filename, exact: true }).click();
  await expect(page.getByRole('button', { name: '파일 편집', exact: true })).toHaveCount(0);
  const editor = page.getByRole('textbox', { name: /inline-edit\.md$/ });
  await expect(editor).toBeVisible();
  await expect(page.getByRole('radio', { name: 'Raw', exact: true })).toBeChecked();
  await editor.fill('# Click to edit\n\n**Unsaved draft**');
  await page.getByRole('button', { name: filename, exact: true }).click();
  await expect(page.getByRole('alertdialog')).toHaveCount(0);
  await expect(page.getByRole('button', { name: '저장', exact: true })).toBeEnabled();
  await page.getByRole('radio', { name: 'Preview', exact: true }).click();
  await page
    .locator('.file-document')
    .getByRole('heading', { name: 'Click to edit', exact: true })
    .click();
  await expect(page.getByRole('radio', { name: 'Raw', exact: true })).toBeChecked();
  await expect(editor).toBeFocused();
  await editor.pressSequentially('!');
  expect(await readFile(join(fixture.projectA, filename), 'utf8')).toBe(source);
  await page.getByRole('button', { name: '저장', exact: true }).click();
  await expect
    .poll(() => readFile(join(fixture.projectA, filename), 'utf8'))
    .toContain('Unsaved draft');
  await writeFile(join(fixture.projectA, filename), source);
});
test('a long unbroken code token wraps inside the Markdown preview', async ({ page }) => {
  await page.getByRole('button', { name: filename, exact: true }).click();
  await page.getByRole('radio', { name: 'Preview', exact: true }).click();
  const code = page.locator('.file-document pre code').first();
  await expect(code).toBeVisible();
  const measure = await code.evaluate((element) => {
    const walker = document.createTreeWalker(element, NodeFilter.SHOW_TEXT);
    const rects: DOMRect[] = [];
    for (let node = walker.nextNode(); node; node = walker.nextNode()) {
      const range = document.createRange();
      range.selectNodeContents(node);
      rects.push(...range.getClientRects());
    }
    return {
      lines: new Set(rects.filter((r) => r.width > 0).map((r) => Math.round(r.y))).size,
      right: Math.max(...rects.map((r) => r.right)),
      width: window.innerWidth,
    };
  });
  expect(measure.lines).toBeGreaterThan(1);
  expect(measure.right).toBeLessThanOrEqual(measure.width);
});
test('oversized files retain bounded read-only viewing without an edit action', async ({
  page,
}) => {
  await page.getByRole('button', { name: 'large-readonly.txt', exact: true }).click();
  await expect(
    page.locator('.file-document').getByText('읽기 전용', { exact: true })
  ).toBeVisible();
  await expect(page.getByRole('textbox', { name: /large-readonly\.txt$/ })).toHaveCount(0);
  await expect(page.getByRole('button', { name: '파일 편집', exact: true })).toHaveCount(0);
  await expect(page.locator('.file-document pre code').first()).toContainText('bounded text');
  await writeFile(
    join(fixture.projectA, 'large-readonly.txt'),
    'Changed read-only page\n' + 'bounded text\n'.repeat(100000)
  );
  await page
    .locator('.file-document')
    .getByRole('button', { name: '새로고침', exact: true })
    .click();
  await expect(page.locator('.file-document pre code').first()).toContainText(
    'Changed read-only page'
  );
});
