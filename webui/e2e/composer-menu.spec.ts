// Purpose: Verify attachment menu behavior and image paste/drop routing.
// Pattern: Browser-to-Go behavior contracts with isolated model fixtures.
// Usage: MECA_UI_DESCRIPTOR points to the opt-in Go browser fixture.
// Related: webui/e2e/fixture.ts, cmd/meca/ui_browser_test.go.

import { readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { expect } from '@playwright/test';
import {
  type BrowserFixture,
  browserFixture,
  connectBrowser,
  create,
  input,
  required,
  send,
  test,
} from '#e2e/fixture';

let fixture: BrowserFixture;
test.beforeAll(async () => {
  fixture = await browserFixture();
});
test.beforeEach(async ({ page }) => {
  await connectBrowser(page, fixture);
});
test('an empty conversation has one composer with an upward image and skill menu', async ({
  page,
}) => {
  await create(page);
  await expect(
    page.getByRole('heading', { name: '어떤 작업을 시작할까요?', exact: true })
  ).toBeVisible();
  await expect(input(page)).toBeEnabled();
  const composer = page.locator('.composer'),
    plus = composer.getByRole('button', { name: '첨부 및 스킬', exact: true });
  await expect(composer.getByRole('button', { name: /^모델 / })).toHaveCount(0);
  await expect(composer.getByRole('button', { name: /받아쓰기/ })).toHaveCount(0);
  await expect(page.getByRole('menuitem', { name: '스킬 목록 보기', exact: true })).toBeHidden();
  await plus.click();
  await expect(plus).toHaveAttribute('aria-expanded', 'true');
  await expect(page.getByRole('menuitem', { name: '스킬 목록 보기', exact: true })).toBeVisible();
  await expect(page.locator('input[type=file]')).toHaveAttribute('accept', 'image/*');
  await expect
    .poll(() => plus.evaluate((node) => getComputedStyle(node).transform))
    .toMatch(/matrix\(0, 1, -1, 0/);
  const panel = page.locator(`[id=${JSON.stringify(await plus.getAttribute('aria-controls'))}]`);
  expect((await panel.boundingBox().then(required)).y).toBeLessThan(
    (await plus.boundingBox().then(required)).y
  );
  await page.keyboard.press('Escape');
  await expect(plus).toHaveAttribute('aria-expanded', 'false');
  await expect(page.getByRole('menuitem', { name: '스킬 목록 보기', exact: true })).toBeHidden();
  await send(page, 'Astryx composition contract');
  await expect(
    page.getByRole('heading', { name: '어떤 작업을 시작할까요?', exact: true })
  ).toBeHidden();
  await expect(page.locator('.history-rows')).toContainText('Astryx composition contract');
});

test('composer accepts image drops and native and explicit clipboard shortcuts without duplicate attachment', async ({
  page,
}) => {
  await create(page);
  const bytes = (await readFile(join(fixture.home, 'fixture.png'))).toString('base64');
  const transfer = await page.evaluateHandle((bytes) => {
    const data = new DataTransfer();
    data.items.add(
      new File([Uint8Array.from(atob(bytes), (c) => c.charCodeAt(0))], 'drop.png', {
        type: 'image/png',
      })
    );
    return data;
  }, bytes);
  await page.locator('.composer').dispatchEvent('drop', { dataTransfer: transfer });
  await expect(page.getByRole('group', { name: /^drop\.png/ })).toHaveCount(1);
  await input(page).dispatchEvent('drop', { dataTransfer: transfer });
  await expect(page.getByRole('group', { name: /^drop\.png/ })).toHaveCount(2);
  await transfer.dispose();
  await input(page).focus();
  await input(page).evaluate((editor, bytes) => {
    const data = new DataTransfer();
    data.items.add(
      new File([Uint8Array.from(atob(bytes), (c) => c.charCodeAt(0))], 'paste.png', {
        type: 'image/png',
      })
    );
    editor.dispatchEvent(
      new ClipboardEvent('paste', { clipboardData: data, bubbles: true, cancelable: true })
    );
  }, bytes);
  await expect(page.getByRole('group', { name: /^paste\.png/ })).toHaveCount(1);
  await page.evaluate((bytes) => {
    const image = new Blob([Uint8Array.from(atob(bytes), (c) => c.charCodeAt(0))], {
      type: 'image/png',
    });
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { read: async () => [{ types: ['image/png'], getType: async () => image }] },
    });
    Object.defineProperty(navigator, 'platform', { configurable: true, value: 'MacIntel' });
  }, bytes);
  await input(page).focus();
  await page.keyboard.press('Alt+KeyV');
  await expect(page.getByRole('group', { name: /^clipboard-1\.png/ })).toHaveCount(1);
  await page.keyboard.press('Control+KeyV');
  await expect(page.getByRole('group', { name: /^clipboard-1\.png/ })).toHaveCount(2);
  await expect(page.getByRole('button', { name: '전송', exact: true })).toBeEnabled();
});
