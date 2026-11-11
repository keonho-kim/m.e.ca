// Purpose: Keep settings, confirmations, workspace selection and image viewers opaque.
// Pattern: Rendered material contracts across native Astryx modal primitives.
// Usage: MECA_UI_DESCRIPTOR points to the isolated embedded UI fixture.
// Related: webui/src/app/styles/theme.ts, webui/src/pages/workspace/ui/inspection/conversation-inspector.tsx.
import { readFile } from 'node:fs/promises';
import { expect, type Locator, test } from '@playwright/test';

async function opaque(layer: Locator) {
  await expect(layer).toBeVisible();
  const material = await layer.evaluate((node) => {
    const style = getComputedStyle(node);
    return {
      background: style.backgroundColor,
      opacity: style.opacity,
      filter: style.backdropFilter,
    };
  });
  expect(material.background).toMatch(/^rgb\(/);
  expect(material.opacity).toBe('1');
  expect(material.filter).toBe('none');
}
test('service modals keep solid surfaces rather than revealing the underlying page', async ({
  page,
}) => {
  test.skip(!process.env.MECA_UI_DESCRIPTOR, 'start the isolated browser fixture first');
  const fixture = JSON.parse(await readFile(process.env.MECA_UI_DESCRIPTOR!, 'utf8'));
  const image = Buffer.from(
    'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=',
    'base64'
  );
  await page.route('**/v1/command', async (route) => {
    if (route.request().postDataJSON().history)
      await route.fulfill({
        json: {
          history: {
            items: [
              {
                sequence: '1',
                kind: 'message',
                role: 'user',
                text: 'Image sample.',
                images: [{ index: 0 }],
              },
            ],
          },
        },
      });
    else await route.continue();
  });
  await page.route('**/images/1/0*', (route) =>
    route.fulfill({ body: image, contentType: 'image/png' })
  );
  await page.goto(`${fixture.url}/ui/`);
  await page
    .getByRole('textbox', { name: '서버 토큰', exact: true })
    .fill((await readFile(fixture.tokenFile, 'utf8')).trim());
  await page.getByRole('button', { name: '연결', exact: true }).click();
  await page
    .locator('.workspace-group')
    .filter({ has: page.locator('.workspace-name', { hasText: 'alpha' }) })
    .getByRole('button', { name: '새 대화', exact: true })
    .click();
  await expect(page.getByText('Image sample.', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: '설정', exact: true }).click();
  const settings = page.getByRole('dialog', { name: '설정', exact: true });
  await opaque(settings);
  await settings.getByRole('button', { name: '실행', exact: true }).click();
  await settings.getByRole('button', { name: '저장', exact: true }).click();
  const confirmation = page.getByRole('alertdialog');
  await opaque(confirmation);
  await confirmation.getByRole('button', { name: '아니오', exact: true }).click();
  await page.keyboard.press('Escape');
  await expect(settings).toHaveCount(0);
  await page.getByRole('button', { name: '워크스페이스 추가', exact: true }).click();
  await opaque(page.getByRole('dialog', { name: '워크스페이스 추가', exact: true }));
  await page.keyboard.press('Escape');
  await page.locator('.history-row .astryx-thumbnail').click();
  await opaque(page.getByRole('dialog'));
});
