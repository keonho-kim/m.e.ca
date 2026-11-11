// Purpose: Verify byte admission, IME transactions and bounded large-document browser editing.
// Pattern: Native editor regression contracts against the isolated engine fixture.
// Usage: MECA_UI_DESCRIPTOR=/tmp/task/connection.json bun run test:e2e editor-boundaries.spec.ts.
// Related: webui/src/shared/ui/code-editor/index.tsx, webui/patches/react-simple-code-editor@0.14.1.patch.
import { readFile, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { expect, type Page, test } from '@playwright/test';

let fixture: { url: string; tokenFile: string; projectA: string };
const limit = 2 * 1024 * 1024;
const source = 'const answer = 42;\nconst result = answer';
const large = '// bounded source line\n'.repeat(44000);
test.beforeAll(async () => {
  test.skip(!process.env.MECA_UI_DESCRIPTOR, 'start the isolated Go fixture first');
  fixture = JSON.parse(await readFile(process.env.MECA_UI_DESCRIPTOR!, 'utf8'));
  for (const [name, text] of Object.entries({
    'boundary-limit.txt': 'x'.repeat(limit - 1),
    'boundary-ime.js': source,
    'boundary-large.txt': large,
    'boundary-mode.js': `${'// ' + 'x'.repeat(96)}\n`.repeat(655),
  })) await writeFile(join(fixture.projectA, name), text);
});
test.beforeEach(async ({ page }) => {
  await page.goto(`${fixture.url}/ui/`);
  await page.getByRole('textbox', { name: '서버 토큰', exact: true })
    .fill((await readFile(fixture.tokenFile, 'utf8')).trim());
  await page.getByRole('button', { name: '연결', exact: true }).click();
  const previous = page.url();
  await page.locator('.workspace-group')
    .filter({ has: page.locator('.workspace-name', { hasText: 'alpha' }) })
    .getByRole('button', { name: '새 대화', exact: true }).click();
  await page.waitForURL((url) => url.href !== previous && url.pathname.includes('/c/session_'));
  await page.getByRole('tab', { name: '파일', exact: true }).click();
});
async function open(page: Page, filename: string) {
  await page.getByRole('button', { name: filename, exact: true }).click();
  const input = page.getByRole('textbox', { name: new RegExp(`${filename.replaceAll('.', '\\.')}$`) });
  await expect(input).toBeVisible();
  return input;
}
async function end(input: ReturnType<Page['getByRole']>) {
  await input.focus();
  await input.evaluate((element) => {
    const textarea = element as HTMLTextAreaElement;
    textarea.setSelectionRange(textarea.value.length, textarea.value.length);
  });
}
const modifier = process.platform === 'darwin' ? 'Meta' : 'Control';

test('two-MiB edits save successfully and excess paste leaves text and undo intact', async ({ page }) => {
  const input = await open(page, 'boundary-limit.txt');
  expect(await input.evaluate((element) => (element as HTMLTextAreaElement).maxLength)).toBe(limit);
  await expect(page.locator('.text-editor pre')).toHaveCount(0);
  const rejected = await input.evaluate((element) => {
    const textarea = element as HTMLTextAreaElement;
    const before = textarea.value;
    textarea.setSelectionRange(0, before.length);
    const paste = (text: string) => {
      const data = new DataTransfer();
      data.setData('text/plain', text);
      const event = new ClipboardEvent('paste', { bubbles: true, cancelable: true, clipboardData: data });
      return !textarea.dispatchEvent(event);
    };
    return { ascii: paste('x'.repeat(2 * 1024 * 1024 + 1)),
      utf8: paste('한'.repeat(Math.floor(2 * 1024 * 1024 / 3) + 1)),
      unchanged: textarea.value === before };
  });
  expect(rejected).toEqual({ ascii: true, utf8: true, unchanged: true });
  await end(input);
  await input.pressSequentially('x');
  expect(await input.evaluate((element) => (element as HTMLTextAreaElement).value.length)).toBe(limit);
  await input.press(`${modifier}+z`);
  expect(await input.evaluate((element) => (element as HTMLTextAreaElement).value.length)).toBe(limit - 1);
  await input.press(`${modifier}+Shift+z`);
  await page.getByRole('button', { name: '저장', exact: true }).click();
  await expect(page.getByRole('button', { name: '저장', exact: true })).toBeDisabled();
  expect((await readFile(join(fixture.projectA, 'boundary-limit.txt'))).length).toBe(limit);
});

test('one completed Korean composition is one undo operation', async ({ page }) => {
  const input = await open(page, 'boundary-ime.js');
  await end(input);
  await input.evaluate(async (element) => {
    const textarea = element as HTMLTextAreaElement;
    const initial = textarea.value;
    const setter = Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, 'value')?.set;
    if (!setter) throw new Error('native textarea setter unavailable');
    textarea.dispatchEvent(new CompositionEvent('compositionstart', { bubbles: true }));
    for (const data of ['ㅎ', '하', '한']) {
      setter.call(textarea, initial + data);
      textarea.setSelectionRange(textarea.value.length, textarea.value.length);
      textarea.dispatchEvent(new InputEvent('input', { bubbles: true, inputType: 'insertCompositionText', isComposing: true, data }));
      await new Promise(requestAnimationFrame);
    }
    textarea.dispatchEvent(new CompositionEvent('compositionend', { bubbles: true, data: '한' }));
  });
  await expect(input).toHaveValue(`${source}한`);
  await input.press(`${modifier}+z`);
  await expect(input).toHaveValue(source);
  await input.press(`${modifier}+Shift+z`);
  await expect(input).toHaveValue(`${source}한`);
  await input.press(`${modifier}+z`);
  const native = await page.context().newCDPSession(page);
  try {
    await end(input);
    for (const text of ['ㅎ', '하'])
      await native.send('Input.imeSetComposition', { text, selectionStart: 1, selectionEnd: 1 });
    await native.send('Input.insertText', { text: '한' });
    await expect(input).toHaveValue(`${source}한`);
    await input.press(`${modifier}+z`);
    await expect(input).toHaveValue(source);
  } finally {
    await native.detach();
  }
});

test('crossing the syntax budget preserves the same input, focus and undo', async ({ page }) => {
  const input = await open(page, 'boundary-mode.js');
  await expect(page.locator('.text-editor pre')).toHaveCount(1);
  const initialLength = await input.evaluate((element) => (element as HTMLTextAreaElement).value.length);
  await end(input);
  await input.evaluate((element) => { element.setAttribute('data-original-input', 'true'); });
  await input.pressSequentially('x'.repeat(50));
  await expect(page.locator('.text-editor pre')).toHaveCount(0);
  await expect(input).toBeFocused();
  await expect(input).toHaveAttribute('data-original-input', 'true');
  await input.press(`${modifier}+z`);
  expect(await input.evaluate((element) => (element as HTMLTextAreaElement).value.length)).toBe(initialLength);
  await expect(page.locator('.text-editor pre')).toHaveCount(1);
});

test('repeated large replacements keep a viewport-sized editor without duplicate source DOM', async ({ page }) => {
  const input = await open(page, 'boundary-large.txt');
  await expect(page.locator('.text-editor pre')).toHaveCount(0);
  const size = await input.boundingBox(), document = await page.locator('.file-document').boundingBox();
  expect(size && document && size.height <= document.height).toBe(true);
  for (let cycle = 0; cycle < 3; cycle++) {
    await end(input);
    await input.pressSequentially('abcdefghijklmnopqrstuvwxyz');
    expect(await input.evaluate((element) => (element as HTMLTextAreaElement).value.length)).toBe(large.length + 26);
    try {
      await input.fill(large, { timeout: 5000 });
    } catch {
      throw new Error('Large document replacement exceeded five seconds');
    }
  }
  expect(await page.locator('.text-editor').locator('*').count()).toBeLessThan(5);
  await expect(page.getByRole('button', { name: '저장', exact: true })).toBeDisabled();
});

test('switching documents releases syntax ranges after every highlighted editor', async ({ page }) => {
  for (let cycle = 0; cycle < 3; cycle++) {
    await open(page, 'boundary-ime.js');
    await expect.poll(() => page.evaluate(() => CSS.highlights.get('astryx-keyword')?.size ?? 0)).toBeGreaterThan(0);
    await open(page, 'boundary-large.txt');
    await expect.poll(() => page.evaluate(() => [...CSS.highlights.values()].reduce((total, ranges) => total + ranges.size, 0))).toBe(0);
  }
});

test('Preview reads preserve the current typing group', async ({ page }) => {
  const input = await open(page, 'README.md');
  const initial = await input.inputValue();
  await end(input);
  await input.pressSequentially('alpha');
  await page.getByRole('radio', { name: 'Preview', exact: true }).click();
  await page.getByRole('radio', { name: 'Raw', exact: true }).click();
  await end(input);
  await input.pressSequentially('beta');
  await input.press(`${modifier}+z`);
  await expect(input).toHaveValue(initial);
});
