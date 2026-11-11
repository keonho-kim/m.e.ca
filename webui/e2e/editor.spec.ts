// Purpose: Verify native textarea editing, isolated highlighting, undo and captured saves.
// Pattern: Browser-to-engine editor replacement contracts.
// Usage: MECA_UI_DESCRIPTOR=/tmp/task/connection.json bun run test:e2e editor.spec.ts.
// Related: webui/src/shared/ui/code-editor/index.tsx, webui/src/shared/ui/code-editor/syntax.ts.
import { readFile, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { expect, type Page, test } from '@playwright/test';

let fixture: { url: string; tokenFile: string; projectA: string };
const source = 'const answer = 42;\nconst result = answer';
const large = '// bounded source line\n'.repeat(44000);
test.beforeAll(async () => {
  test.skip(!process.env.MECA_UI_DESCRIPTOR, 'start the isolated Go fixture first');
  fixture = JSON.parse(await readFile(process.env.MECA_UI_DESCRIPTOR!, 'utf8'));
  for (const [name, text] of Object.entries({
    'editor-word.js': source,
    'editor-save.js': source,
    'editor-large.txt': large,
    'editor-literal.html': '<img src=x onerror="document.body.dataset.editorInjected=1">',
  }))
    await writeFile(join(fixture.projectA, name), text);
});
test.beforeEach(async ({ page }) => {
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
  await page.getByRole('tab', { name: '파일', exact: true }).click();
});
async function open(page: Page, filename: string) {
  await page.getByRole('button', { name: filename, exact: true }).click();
  const input = page.getByRole('textbox', {
    name: new RegExp(`${filename.replaceAll('.', '\\.')}$`),
  });
  await expect(input).toBeVisible();
  expect(await input.evaluate((node) => node.tagName)).toBe('TEXTAREA');
  return input;
}
async function atEnd(input: ReturnType<Page['getByRole']>) {
  await input.evaluate((element) => {
    const textarea = element as HTMLTextAreaElement;
    textarea.setSelectionRange(textarea.value.length, textarea.value.length);
  });
}
test('syntax uses ranges without token DOM and the first word edit remains undoable', async ({
  page,
}) => {
  const editor = await open(page, 'editor-word.js');
  await expect(editor).toHaveValue(source);
  const inputBounds = await editor.boundingBox(),
    documentBounds = await page.locator('.file-document').boundingBox();
  expect(inputBounds!.height / documentBounds!.height).toBeGreaterThan(0.75);
  await expect
    .poll(() =>
      page.locator('.text-editor pre').evaluate((pre) => {
        const highlight = CSS.highlights.get('astryx-keyword');
        return [...(highlight ?? [])].filter((range) => pre.contains(range.startContainer)).length;
      })
    )
    .toBeGreaterThan(0);
  expect(await page.locator('.text-editor').locator('*').count()).toBeLessThan(10);
  await atEnd(editor);
  await editor.pressSequentially('X');
  await expect(editor).toHaveValue(`${source}X`);
  const modifier = process.platform === 'darwin' ? 'Meta' : 'Control';
  await editor.press(`${modifier}+z`);
  await expect(editor).toHaveValue(source);
  await editor.press(`${modifier}+Shift+z`);
  await expect(editor).toHaveValue(`${source}X`);
});
test('HTML source is literal and IME Enter is left to the browser', async ({ page }) => {
  const editor = await open(page, 'editor-literal.html');
  await expect(editor).toHaveValue(/<img src=x onerror=/);
  await expect(page.locator('.text-editor img')).toHaveCount(0);
  expect(await page.evaluate(() => document.body.dataset.editorInjected)).toBeUndefined();
  const result = await editor.evaluate((element) => {
    const input = element as HTMLTextAreaElement;
    const before = input.value;
    const allowed = input.dispatchEvent(
      new KeyboardEvent('keydown', {
        key: 'Enter',
        keyCode: 229,
        isComposing: true,
        bubbles: true,
        cancelable: true,
      })
    );
    return { allowed, unchanged: before === input.value };
  });
  expect(result).toEqual({ allowed: true, unchanged: true });
});
test('edits made while a save reply is delayed remain dirty', async ({ page }) => {
  const editor = await open(page, 'editor-save.js');
  await editor.fill('const saved = 1;');
  let applied!: () => void, release!: () => void;
  const saved = new Promise<void>((resolve) => {
    applied = resolve;
  });
  const delayed = new Promise<void>((resolve) => {
    release = resolve;
  });
  await page.route('**/v1/command', async (route) => {
    if (!route.request().postDataJSON()?.editorSave) {
      await route.continue();
      return;
    }
    const response = await route.fetch();
    applied();
    await delayed;
    await route.fulfill({ response });
  });
  try {
    await page.getByRole('button', { name: '저장', exact: true }).click();
    await saved;
    await atEnd(editor);
    await editor.pressSequentially(' // later');
    release();
    await expect(page.getByRole('button', { name: '저장', exact: true })).toBeEnabled();
    await expect(editor).toHaveValue('const saved = 1; // later');
    expect(await readFile(join(fixture.projectA, 'editor-save.js'), 'utf8')).toBe(
      'const saved = 1;'
    );
  } finally {
    release();
    await page.unroute('**/v1/command');
  }
});
test('many-line documents remain editable with bounded DOM and working undo', async ({ page }) => {
  const editor = await open(page, 'editor-large.txt');
  await expect(editor).toHaveValue(large);
  expect(await page.locator('.text-editor').locator('*').count()).toBeLessThan(10);
  await atEnd(editor);
  await editor.pressSequentially('X');
  await expect(editor).toHaveValue(`${large}X`);
  const modifier = process.platform === 'darwin' ? 'Meta' : 'Control';
  await editor.press(`${modifier}+z`);
  await expect(editor).toHaveValue(large);
});
