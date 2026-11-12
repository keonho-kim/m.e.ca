// Purpose: Guard deferred reader loading and opaque, reduced-motion-aware pane and dialog transitions.
// Pattern: Browser contracts over the real embedded build with controlled model output.
// Usage: MECA_UI_DESCRIPTOR points to the isolated Go UI fixture.
// Related: webui/src/app/styles/motion.css, webui/src/pages/workspace/ui/chat/chat.tsx.

import { expect, type Page } from '@playwright/test';
import { type BrowserFixture, browserFixture, connectBrowser, test } from '#e2e/fixture';

let fixture: BrowserFixture;
test.beforeAll(async () => {
  fixture = await browserFixture();
});
async function emptyConversation(page: Page) {
  await connectBrowser(page, fixture);
  await page
    .locator('.workspace-group')
    .filter({ has: page.locator('.workspace-name', { hasText: 'alpha' }) })
    .getByRole('button', { name: '새 대화', exact: true })
    .click();
  await expect(
    page.getByRole('heading', { name: '어떤 작업을 시작할까요?', exact: true })
  ).toBeVisible();
}
test('empty conversations defer rich reader code until a response actually needs it', async ({
  page,
}) => {
  const readerCode: Promise<boolean>[] = [];
  page.on('response', (response) => {
    if (!response.url().endsWith('.js')) return;
    readerCode.push(response.text().then((text) => text.includes('meca-render-pages')));
  });
  await emptyConversation(page);
  expect((await Promise.all(readerCode)).some(Boolean)).toBe(false);
  await page
    .getByRole('combobox', { name: '작업 입력', exact: true })
    .fill('Deferred reader fixture');
  await page.locator('.composer').getByRole('button', { name: '전송', exact: true }).click();
  await expect(page.locator('.completion-summary')).toBeVisible();
  await expect(page.locator('.history-rows')).toContainText('Completed: Deferred reader fixture');
  expect((await Promise.all(readerCode)).some(Boolean)).toBe(true);
});

test('dialog arrival moves a solid surface and reduced motion cancels spatial transitions', async ({
  page,
}) => {
  await page.emulateMedia({ reducedMotion: 'no-preference' });
  await emptyConversation(page);
  const cdp = await page.context().newCDPSession(page);
  await cdp.send('Animation.enable');
  await cdp.send('Animation.setPlaybackRate', { playbackRate: 0.1 });
  try {
    await page.getByRole('button', { name: '설정', exact: true }).click();
    const dialog = page.getByRole('dialog', { name: '설정', exact: true });
    await expect(dialog).toBeVisible();
    const motion = await dialog.evaluate((node) => {
      const style = getComputedStyle(node);
      const animation = node
        .getAnimations()
        .find(
          (value) => value instanceof CSSAnimation && value.animationName === 'meca-dialog-enter'
        );
      if (!animation) return { name: style.animationName, samples: [], duration: 0 };
      animation.pause();
      const duration = Number(animation.effect?.getTiming().duration);
      const samples = [0, 0.5, 1].map((progress) => {
        animation.currentTime = duration * progress;
        return {
          opacity: getComputedStyle(node).opacity,
          background: getComputedStyle(node).backgroundColor,
        };
      });
      return { name: style.animationName, samples, duration };
    });
    expect(motion.name).toBe('meca-dialog-enter');
    expect(motion.duration).toBeGreaterThan(0);
    expect(motion.duration).toBeLessThanOrEqual(300);
    expect(motion.samples).toHaveLength(3);
    for (const sample of motion.samples) {
      expect(sample.opacity).toBe('1');
      expect(sample.background).toMatch(/^rgb\(/);
    }
    await page.emulateMedia({ reducedMotion: 'reduce' });
    await expect
      .poll(() => dialog.evaluate((node) => getComputedStyle(node).animationName))
      .toBe('none');
    await page.keyboard.press('Escape');
    await page.getByRole('tab', { name: '파일', exact: true }).click();
    const pane = page.locator('.files-workspace');
    await expect(pane).toBeVisible();
    expect(await pane.evaluate((node) => getComputedStyle(node).animationName)).toBe('none');
    await page.emulateMedia({ reducedMotion: 'no-preference' });
    await page.getByRole('tab', { name: '스킬', exact: true }).click();
    await expect(page.locator('.skills-workspace')).toBeVisible();
    expect(
      await page
        .locator('.skills-workspace')
        .evaluate((node) => getComputedStyle(node).animationName)
    ).toBe('meca-pane-enter');
  } finally {
    await cdp.send('Animation.setPlaybackRate', { playbackRate: 1 });
    await cdp.detach();
  }
});
