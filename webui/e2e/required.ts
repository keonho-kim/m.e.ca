// Purpose: Reject success when a selected browser smoke check never ran.
// Pattern: Playwright result gate.
// Usage: The browser configuration registers this reporter alongside line output.
// Related: webui/playwright.config.ts, webui/package.json, .github/workflows/release.yml.
import type { FullConfig, FullResult, Reporter, Suite, TestCase } from '@playwright/test/reporter';

export default class RequiredChecks implements Reporter {
  private smoke: TestCase[] = [];

  onBegin(_config: FullConfig, suite: Suite) {
    this.smoke = suite.allTests().filter((test) => test.tags.includes('@smoke'));
  }

  async onEnd(result: FullResult) {
    if (result.status !== 'passed' || process.argv.includes('--list')) return;
    const missing = this.smoke.filter((test) => test.outcome() === 'skipped');
    if (!missing.length) return;
    for (const test of missing) console.error(`Required browser check did not run: ${test.title}`);
    return { status: 'failed' as const };
  }
}
