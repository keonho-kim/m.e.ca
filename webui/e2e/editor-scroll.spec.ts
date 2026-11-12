// Purpose: Verify complete editable documents remain reachable through real wheel and caret scrolling.
// Pattern: Browser editor geometry regression over the actual embedded server.
// Usage: MECA_UI_DESCRIPTOR=/tmp/task/connection.json bun run test:e2e editor-scroll.spec.ts.
// Related: webui/src/shared/ui/code-editor/index.tsx, webui/src/pages/workspace/ui/files/file-document.tsx.

import { writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { expect } from '@playwright/test';
import { type BrowserFixture, browserFixture, connectBrowser, required, test } from '#e2e/fixture';

let fixture: BrowserFixture;
test.beforeAll(async () => {
  fixture = await browserFixture();
  for (const extension of ['ts', 'md'])
    await writeFile(
      join(fixture.projectA, `scroll-check.${extension}`),
      Array.from({ length: 800 }, (_, index) => `line ${index}: complete document`).join('\n')
    );
});
test.beforeEach(async ({ page }) => {
  await connectBrowser(page, fixture);
  await page.getByRole('tab', { name: '파일', exact: true }).click();
});

for (const extension of ['ts', 'md']) {
  test(`raw ${extension} document scrolls to its last line`, async ({ page }) => {
    const filename = `scroll-check.${extension}`;
    await page.getByRole('button', { name: filename, exact: true }).click();
    const editor = page.getByRole('textbox', { name: new RegExp(`scroll-check\\.${extension}$`) });
    await expect(editor).toBeVisible();
    const bounds = await editor.boundingBox().then(required);
    expect(bounds).not.toBeNull();
    await page.mouse.move(bounds.x + bounds.width / 2, bounds.y + 150);
    await page.mouse.wheel(0, 16000);
    await expect
      .poll(() =>
        editor.evaluate((node) => {
          let top = 0;
          for (let element: Element | null = node; element; element = element.parentElement)
            top = Math.max(top, element.scrollTop);
          return top;
        })
      )
      .toBeGreaterThan(1000);
    await editor.press('ControlOrMeta+End');
    expect(await editor.inputValue()).toContain('line 799: complete document');
    expect(
      await page.locator('.file-document').evaluate((node) => node.scrollWidth <= node.clientWidth)
    ).toBe(true);
  });
}
