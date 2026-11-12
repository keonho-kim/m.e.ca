// Purpose: Bound hidden chat DOM updates while retaining terminal delivery and readable telemetry.
// Pattern: Browser-to-engine burst regression with explicit provider admission.
// Usage: MECA_UI_DESCRIPTOR points to the isolated Go UI fixture.
// Related: cmd/meca/ui_burst_provider_test.go, webui/src/pages/workspace/model/presentation.ts.
import { expect } from '@playwright/test';
import { browserFixture, connectBrowser, create, send, test } from '#e2e/fixture';

test('opaque settings coalesce live text but terminal measurements arrive immediately', async ({
  page,
}) => {
  const fixture = await browserFixture();
  await connectBrowser(page, fixture);
  await create(page);
  await send(page, 'BURST_FIXTURE');
  await expect(page.locator('.live-output')).toContainText('Burst ready.');
  await page.getByRole('button', { name: '설정', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: '설정', exact: true });
  await expect(dialog).toBeVisible();
  await expect(page.locator('.live-output')).not.toContainText('Background presentation update.');
  await page.evaluate(() => {
    const node = document.querySelector('.live-output');
    if (!node) throw Error('live output missing');
    const probe = {
      mutations: 0,
      updates: 0,
      text: node.textContent,
      observer: new MutationObserver((records) => {
        probe.mutations += records.length;
        if (probe.text !== node.textContent) {
          probe.text = node.textContent;
          probe.updates++;
        }
      }),
    };
    probe.observer.observe(node, { subtree: true, childList: true, characterData: true });
    Object.assign(window, { presentationProbe: probe });
  });
  try {
    await page.request.post(`${fixture.provider}/test/start-burst`);
    await expect(page.locator('.completion-summary')).toContainText('입력 12,000');
    await expect(page.locator('.completion-summary')).toContainText('출력 6,400');
    const counts = await page.evaluate(() => {
      const probe = (
        window as unknown as {
          presentationProbe: { mutations: number; updates: number; observer: MutationObserver };
        }
      ).presentationProbe;
      probe.observer.disconnect();
      return { mutations: probe.mutations, updates: probe.updates };
    });
    expect(counts.mutations).toBeLessThan(60);
    expect(counts.updates).toBeLessThanOrEqual(8);
    await expect(dialog).toBeVisible();
    await dialog.getByRole('button', { name: '닫기', exact: true }).click();
    await expect(page.locator('.history-rows')).toContainText('Completed burst.');
    const typography = await page
      .locator('.completion-summary .astryx-text')
      .first()
      .evaluate((node) => {
        const style = getComputedStyle(node);
        return { size: parseFloat(style.fontSize), numbers: style.fontVariantNumeric };
      });
    expect(typography.size).toBeGreaterThanOrEqual(14);
    expect(typography.numbers).toContain('tabular-nums');
  } finally {
    await page.evaluate(() => {
      const probe = (window as unknown as { presentationProbe?: { observer: MutationObserver } })
        .presentationProbe;
      probe?.observer.disconnect();
    });
  }
});
