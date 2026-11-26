// Purpose: Verify native textarea editing, isolated highlighting, undo and captured saves.
// Pattern: Browser-to-engine editor replacement contracts.
// Usage: Run through the Playwright browser suite.
// Related: webui/src/shared/ui/code-editor/index.tsx, webui/src/shared/ui/code-editor/syntax.ts.

import { readFile, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { expect, type Page } from '@playwright/test';
import { connectBrowser, create, required, test } from '#e2e/fixture';

const source = 'const answer = 42;\nconst result = answer';
const large = '// bounded source line\n'.repeat(44000);
test.beforeEach(async ({ page, server: fixture }) => {
  for (const [name, text] of Object.entries({
    'editor-word.js': source,
    'editor-save.js': source,
    'editor-large.ts': large,
    'editor-literal.html': '<img src=x onerror="document.body.dataset.editorInjected=1">',
  }))
    await writeFile(join(fixture.projectA, name), text);
  await connectBrowser(page, fixture);
  await create(page);
});
async function open(page: Page, filename: string) {
  await page.getByRole('tab', { name: '파일', exact: true }).click();
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
  const inputBounds = await editor.boundingBox().then(required),
    documentBounds = await page.locator('.file-document').boundingBox().then(required);
  expect(inputBounds.height / documentBounds.height).toBeGreaterThan(0.75);
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
test('edits made while a save reply is delayed remain dirty', { tag: '@smoke' }, async ({
  page,
  server: fixture,
}) => {
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
test('a many-line file opens once and remains editable with bounded DOM and undo', async ({
  page,
}) => {
  const reads = { directory: 0, editor: 0, preview: 0 };
  page.on('request', (request) => {
    if (!request.url().endsWith('/v1/command')) return;
    const body = request.postDataJSON();
    if (body?.directory) reads.directory++;
    if (body?.editorRead) reads.editor++;
    if (body?.previewFile) reads.preview++;
  });
  const editor = await open(page, 'editor-large.ts');
  await expect(editor).toHaveValue(large);
  expect(await page.locator('.file-document').locator('*').count()).toBeLessThan(2000);
  expect(await page.locator('.text-editor').locator('*').count()).toBeLessThan(10);
  await atEnd(editor);
  await editor.pressSequentially('X');
  await expect(editor).toHaveValue(`${large}X`);
  const modifier = process.platform === 'darwin' ? 'Meta' : 'Control';
  await editor.press(`${modifier}+z`);
  await expect(editor).toHaveValue(large);
  expect(reads).toEqual({ directory: 1, editor: 1, preview: 0 });
});
