// Purpose: Keep actual parallel waves together across durable and live handoff.
// Pattern: Pure projection regression contracts.
// Usage: bun test src/pages/workspace/model/tool-batches.test.ts.
// Related: webui/src/pages/workspace/model/history-rows.ts, webui/src/pages/workspace/model/chat-events.ts.
import { expect, test } from 'bun:test';
import { liveActivity } from '@/pages/workspace/model/chat-events';
import { historyRows } from '@/pages/workspace/model/history-rows';
import type { EventJson } from '@/shared/api';

const event = (executionId: string, batchId = 'wave', agent = 'peer'): EventJson => ({
  kind: 'tool_started',
  runId: 'run',
  agent,
  tool: { executionId, batchId, name: 'read_file' },
});
test('parallel lifecycle updates replace their own entry beneath one agent heading', () => {
  const first = event('one'),
    second = event('two');
  const finished = { ...first, kind: 'tool_succeeded' };
  const rows = historyRows(
    [first, second, finished].map((event, i) => ({ sequence: String(i + 1), event }))
  );
  expect(rows).toHaveLength(1);
  expect('item' in rows[0]! && rows[0].tools?.map((item) => item.event)).toEqual([
    finished,
    second,
  ]);
});
test('agents, runs, serial calls and later waves remain independent', () => {
  const entries = [
    event('1'),
    event('2', 'wave', 'other'),
    event('3', 'next'),
    event('4', ''),
    { ...event('5'), runId: 'next-run' },
  ];
  expect(historyRows(entries.map((event) => ({ event })))).toHaveLength(5);
});
test('the first saved batch member owns all remaining live members without duplicating a heading', () => {
  const first = event('1'),
    second = event('2'),
    other = event('3', 'next');
  expect(liveActivity([first, second, other], [{ event: first }])).toEqual([other]);
});
