// Purpose: Verify the completed response reports measured throughput for zero output.
// Pattern: Deterministic presentation regression test.
// Usage: bun test src/pages/workspace/ui/chat/format-statistics.test.ts.
// Related: webui/src/pages/workspace/ui/chat/format-statistics.ts, internal/tui/transcript/run_summary.go.
import { expect, test } from 'bun:test';
import { throughput } from '@/pages/workspace/ui/chat/format-statistics';

test('completed measurements report zero TPS when generation took time without output', () => {
  const rate = throughput({ generationMillis: '4000', usage: { input: '13' } });
  expect(rate).toBe('0.0 tok/s');
});
