// Purpose: Verify the toolbar language menu, persisted themes and reduced-motion transitions.
// Pattern: Real browser presentation and command contracts.
// Usage: MECA_UI_DESCRIPTOR points to the isolated Go browser fixture.
// Related: webui/src/pages/workspace/ui/navigation/language-menu.tsx, webui/src/app/styles/motion.css.

import { expect } from '@playwright/test';
import { browserFixture, connectBrowser, create, send, test } from '#e2e/fixture';

test.beforeEach(async ({ page }) => {
  await connectBrowser(page, await browserFixture());
});
test('light is the default and theme changes preserve opacity and the saved preference', async ({
  page,
}) => {
  await page.emulateMedia({ colorScheme: 'dark', reducedMotion: 'no-preference' });
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
  const theme = page.getByRole('button', { name: '화면 테마', exact: true });
  for (const mode of ['dark', 'light']) {
    await theme.click();
    await expect(page.locator('html')).toHaveAttribute('data-theme', mode);
    expect(
      await page.locator('.astryx-app-shell').evaluate((node) => getComputedStyle(node).opacity)
    ).toBe('1');
  }
  await theme.click();
  await page.reload();
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await page.getByRole('button', { name: '화면 테마', exact: true }).click();
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
  await page.getByRole('button', { name: '설정', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: '설정', exact: true });
  expect(await dialog.evaluate((node) => getComputedStyle(node).animationName)).toBe('none');
  await expect(dialog.getByRole('button', { name: '일반', exact: true })).toHaveCount(0);
  await expect(dialog.getByRole('button', { name: '연결 종료', exact: true })).toHaveCount(0);
});
test('language changes in the toolbar and !language opens that same menu', async ({ page }) => {
  await create(page);
  await send(page, '!language');
  await expect(page.getByRole('menuitem', { name: '한국어', exact: true })).toHaveAttribute(
    'aria-current',
    'true'
  );
  await expect(page.getByRole('menuitemradio')).toHaveCount(0);
  const englishChoice = page.getByRole('menuitem', { name: 'English', exact: true });
  await englishChoice.focus();
  await englishChoice.press('Enter');
  await expect(page.getByRole('tab', { name: 'Conversation', exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Language', exact: true }).click();
  const english = page.getByRole('menuitem', { name: 'English', exact: true });
  await expect(english).toHaveAttribute('aria-current', 'true');
  await expect(english.locator('svg')).toHaveCount(1);
  await page.getByRole('menuitem', { name: '한국어', exact: true }).click();
  await expect(page.getByRole('tab', { name: '대화', exact: true })).toBeVisible();
  await page.reload();
  await page.getByRole('button', { name: '언어', exact: true }).click();
  await expect(page.getByRole('menuitem', { name: '한국어', exact: true })).toHaveAttribute(
    'aria-current',
    'true'
  );
  await page.keyboard.press('Escape');
  await expect(page.getByRole('menu')).toHaveCount(0);
});
test('sidebar collapse and expansion animate their clipped edge without per-frame width layout', async ({
  page,
}) => {
  await page.emulateMedia({ reducedMotion: 'no-preference' });
  const nav = page.getByRole('navigation', { name: '워크스페이스', exact: true });
  for (const label of ['사이드바 접기', '사이드바 펼치기']) {
    await nav.getByRole('button', { name: label, exact: true }).click();
    const frames = await nav.evaluate(async (node) => {
      const values: { width: number; clip: string }[] = [];
      for (let index = 0; index < 30; index++) {
        await new Promise(requestAnimationFrame);
        values.push({
          width: Math.round(node.getBoundingClientRect().width),
          clip: [...document.querySelectorAll('.astryx-app-shell-sidenav')]
            .map((rail) => getComputedStyle(rail).clipPath)
            .join('|'),
        });
      }
      return values;
    });
    expect(new Set(frames.map((frame) => frame.width)).size).toBeLessThanOrEqual(2);
    expect(new Set(frames.map((frame) => frame.clip)).size).toBeGreaterThan(2);
  }
});
