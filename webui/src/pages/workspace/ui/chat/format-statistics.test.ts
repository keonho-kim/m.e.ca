// Purpose: Verify numeric-only completion telemetry and measured throughput for zero output.
// Pattern: Deterministic presentation regression test.
// Usage: bun test src/pages/workspace/ui/chat/format-statistics.test.ts.
// Related: webui/src/pages/workspace/ui/chat/format-statistics.ts, internal/tui/transcript/run_summary.go.
import { expect, test } from 'bun:test';
import {
  completionMeasurements,
  throughput,
  tokenSummary,
} from '@/pages/workspace/ui/chat/format-statistics';

test('completed measurements report zero TPS when generation took time without output', () => {
  const rate = throughput({ generationMillis: '4000', usage: { input: '13' } });
  expect(rate).toBe('0.0 tok/s');
});

test('large counts retain int64 precision and use readable grouping', () => {
  const parts = tokenSummary({ input: '9223372036854775807', output: '1234567' }, (key) => key);
  expect(parts.join(' ')).toContain('9,223,372,036,854,775,807');
  expect(parts.join(' ')).toContain('1,234,567');
  expect(throughput({ generationMillis: '1000', usage: { output: '1200' } })).toBe('1,200.0 tok/s');
});

test('numeric statistics never echo arbitrary API strings', () => {
  const text = completionMeasurements(
    {
      elapsedMillis: 'untrusted elapsed narrative',
      generationMillis: '1000',
      usage: { input: 'AGENT_RESPONSE', output: 'ANOTHER_RESPONSE', cost: Infinity },
    },
    () => '소요 시간'
  ).join(' ');
  expect(text).not.toContain('AGENT_RESPONSE');
  expect(text).not.toContain('ANOTHER_RESPONSE');
  expect(text).not.toContain('narrative');
  expect(text).not.toContain('Infinity');
});
