// Purpose: Preserve readable sidebar labels throughout light and dark theme changes.
// Pattern: Browser contrast regression over sampled CSS transition progress.
// Usage: MECA_UI_DESCRIPTOR points to the isolated embedded UI fixture.
// Related: webui/src/app/styles/motion.css, webui/e2e/appearance.spec.ts.
import { expect } from '@playwright/test';
import { browserFixture, connectBrowser, create, test } from '#e2e/fixture';

test('project and file sidebars keep text contrast throughout theme changes', async ({ page }) => {
  await connectBrowser(page, await browserFixture());
  await page.emulateMedia({ reducedMotion: 'no-preference' });
  await create(page);
  await page.getByRole('tab', { name: '파일', exact: true }).click();
  await expect(page.locator('.directory-column')).toBeVisible();
  const theme = page.getByRole('button', { name: '화면 테마', exact: true });
  for (const mode of ['dark', 'light']) {
    await theme.click();
    await expect(page.locator('html')).toHaveAttribute('data-theme', mode);
    const samples = await page.evaluate(() => {
      const labels = [
        document.querySelector('.workspace-name'),
        document.querySelector('.directory-column .astryx-text'),
      ];
      if (labels.some((label) => !label)) throw Error('Expected both sidebar labels');
      const canvas = document.createElement('canvas');
      canvas.width = canvas.height = 1;
      const ctx = canvas.getContext('2d');
      if (!ctx) throw Error('Expected browser color resolution');
      const luminance = () => {
        const [r = 0, g = 0, b = 0] = ctx.getImageData(0, 0, 1, 1).data;
        const linear = [r, g, b].map((v) => {
          const channel = v / 255;
          return channel <= 0.04045 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2.4;
        });
        return (linear[0] ?? 0) * 0.2126 + (linear[1] ?? 0) * 0.7152 + (linear[2] ?? 0) * 0.0722;
      };
      const transitions = document.getAnimations().filter((a) => a instanceof CSSTransition);
      for (const transition of transitions) transition.pause();
      try {
        return [0, 0.25, 0.5, 0.75, 1].map((progress) => {
          for (const transition of transitions)
            transition.currentTime = Number(transition.effect?.getTiming().duration) * progress;
          return labels.map((label) => {
            if (!label) throw Error('Missing sidebar label');
            const layers: string[] = [];
            for (let node: Element | null = label; node; node = node.parentElement)
              layers.push(getComputedStyle(node).backgroundColor);
            ctx.fillStyle =
              getComputedStyle(document.documentElement).colorScheme === 'dark' ? '#000' : '#fff';
            ctx.fillRect(0, 0, 1, 1);
            for (const color of layers.reverse()) {
              ctx.fillStyle = color;
              ctx.fillRect(0, 0, 1, 1);
            }
            const background = luminance();
            ctx.fillStyle = getComputedStyle(label).color;
            ctx.fillRect(0, 0, 1, 1);
            const foreground = luminance();
            return (
              (Math.max(background, foreground) + 0.05) / (Math.min(background, foreground) + 0.05)
            );
          });
        });
      } finally {
        for (const transition of transitions) transition.finish();
      }
    });
    for (const [progress, contrasts] of samples.entries())
      for (const contrast of contrasts)
        expect(contrast, `${mode} theme, transition sample ${progress}`).toBeGreaterThanOrEqual(
          4.5
        );
  }
});
