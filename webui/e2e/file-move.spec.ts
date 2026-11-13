// Purpose: Verify non-drag moves preserve directory revisions across destination pages.
// Pattern: Native filesystem contract through the accessible move dialog.
// Usage: MECA_UI_DESCRIPTOR points to the isolated browser fixture.
// Related: webui/src/pages/workspace/ui/files/move-dialog.tsx, internal/fileview/entry_mutation.go.
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { expect } from '@playwright/test';
import { browserFixture, connectBrowser, create, test } from '#e2e/fixture';

test('the move menu reaches a second destination page and moves the original file once', async ({
  page,
}) => {
  const fixture = await browserFixture();
  const folder = `move-folders-${crypto.randomUUID().slice(0, 8)}`;
  const name = `move-source-${crypto.randomUUID().slice(0, 8)}.txt`;
  await mkdir(join(fixture.projectA, folder));
  await Promise.all(
    Array.from({ length: 201 }, (_, index) =>
      mkdir(join(fixture.projectA, folder, `destination-${String(index).padStart(3, '0')}`))
    )
  );
  await writeFile(join(fixture.projectA, name), 'Native move fixture');
  await connectBrowser(page, fixture);
  await create(page);
  await page.getByRole('tab', { name: '파일', exact: true }).click();
  await page.getByRole('button', { name: `파일 작업 · ${name}`, exact: true }).click();
  await page.getByRole('menuitem', { name: '이동…', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: '이동…', exact: true });
  await dialog.getByRole('textbox', { name: '대상 폴더', exact: true }).fill(folder);
  await dialog.getByRole('button', { name: '하위 폴더 열기', exact: true }).click();
  await dialog.getByRole('button', { name: '다음 페이지', exact: true }).click();
  await dialog.getByRole('button', { name: 'destination-200', exact: true }).click();
  await dialog.getByRole('button', { name: '이동…', exact: true }).click();
  await expect(dialog).toHaveCount(0);
  await expect
    .poll(() =>
      readFile(join(fixture.projectA, folder, 'destination-200', name), 'utf8').catch(() => '')
    )
    .toBe('Native move fixture');
});
