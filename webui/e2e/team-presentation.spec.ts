// Purpose: Keep real team contributions and public speech out of measurement and notice surfaces.
// Pattern: Browser-to-engine contract regression through native tools and persisted history.
// Usage: MECA_UI_DESCRIPTOR points to TestUIBrowserFixture.
// Related: cmd/meca/ui_team_provider_test.go, webui/src/pages/workspace/ui/chat/team-work-row.tsx.
import { expect } from '@playwright/test';
import { browserFixture, connectBrowser, create, send, test } from '#e2e/fixture';

test('team work and final speech retain their own rows through persistence and reconnect', async ({
  page,
}) => {
  await connectBrowser(page, await browserFixture());
  await create(page);
  const team = page.getByRole('button', { name: '팀 모드', exact: true });
  await team.click();
  await expect(team).toHaveAttribute('aria-pressed', 'true');
  await send(page, 'TEAM_PRESENTATION_FIXTURE: verify the API together and report.');
  await expect(page.locator('.completion-row')).toHaveCount(1);
  const contributions = page
    .locator('.team-contribution')
    .filter({ hasText: 'TEAM_WORK_PROSE_CANARY' });
  await expect(contributions).toHaveCount(2);
  await expect(page.locator('.chat-content')).toContainText('TEAM_FINAL_RESPONSE_CANARY');
  await expect(page.locator('.completion-row')).not.toContainText(/CANARY|TEAM_PRESENTATION/);
  await expect(
    page.locator('.chat-notice').filter({ hasText: /CANARY|TEAM_PRESENTATION/ })
  ).toHaveCount(0);
  await expect(page.locator('.live-output .team-contribution')).toHaveCount(0);
  await page.reload();
  await expect(contributions).toHaveCount(2);
  await expect(page.locator('.completion-row')).toHaveCount(1);
  await expect(page.locator('.completion-row')).not.toContainText(/CANARY|TEAM_PRESENTATION/);
});
