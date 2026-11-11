// Purpose: Revalidate the open editor without confusing sibling writes or acknowledged saves with conflicts.
// Pattern: Real browser and isolated filesystem observer regression.
// Usage: MECA_UI_DESCRIPTOR=... bun run test:e2e e2e/editor-watch.spec.ts.
// Related: webui/src/pages/workspace/model/file-events.ts, webui/src/pages/workspace/model/file-document.ts.
import { readFile, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { expect, test } from '@playwright/test';
test('one watcher follows documents without full reads or false conflict banners', async ({
  page,
}) => {
  test.skip(!process.env.MECA_UI_DESCRIPTOR, 'start the isolated browser fixture first');
  const fixture = JSON.parse(await readFile(process.env.MECA_UI_DESCRIPTOR!, 'utf8'));
  const name = 'watch-document.txt';
  await writeFile(join(fixture.projectA, name), 'original');
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
  let watchers = 0,
    reads = 0;
  page.on('request', (request) => {
    if (request.url().includes('/v1/files/events?')) watchers++;
    if (request.url().endsWith('/v1/command') && request.postDataJSON()?.editorRead) reads++;
  });
  await page.getByRole('tab', { name: '파일', exact: true }).click();
  await page.getByRole('button', { name, exact: true }).click();
  const editor = page.getByRole('textbox', { name: /watch-document\.txt$/ });
  await expect(editor).toHaveValue('original');
  const baselineWatchers = watchers,
    baselineReads = reads;
  // Observe the resulting directory refresh instead of sleeping for fsnotify.
  const siblingRefresh = page.waitForResponse(
    (response) =>
      response.url().endsWith('/v1/command') && !!response.request().postDataJSON()?.directory
  );
  await writeFile(join(fixture.projectA, 'watch-sibling.txt'), 'sibling');
  await siblingRefresh;
  await expect(
    page.getByText('파일이 외부에서 변경되었습니다. 새로고침으로 현재 내용을 확인하세요.', {
      exact: true,
    })
  ).toHaveCount(0);
  await editor.fill('saved');
  const saved = page.waitForResponse(
    (response) =>
      response.url().endsWith('/v1/command') && !!response.request().postDataJSON()?.editorSave
  );
  await page.getByRole('button', { name: '저장', exact: true }).click();
  await saved;
  await expect(page.getByRole('button', { name: '저장', exact: true })).toBeDisabled();
  const probed = page.waitForResponse(
    (response) =>
      response.url().endsWith('/v1/command') && !!response.request().postDataJSON()?.editorProbe
  );
  await writeFile(join(fixture.projectA, name), 'external replacement');
  await probed;
  await expect(
    page.getByText('파일이 외부에서 변경되었습니다. 새로고침으로 현재 내용을 확인하세요.', {
      exact: true,
    })
  ).toBeVisible();
  expect(watchers).toBe(baselineWatchers);
  expect(reads).toBe(baselineReads);
  await expect(editor).toHaveValue('saved');
});
