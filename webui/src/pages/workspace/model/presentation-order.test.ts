// Purpose: Verify that a later persisted tool cannot precede an earlier active generation.
// Pattern: Pure mixed-timeline regression.
// Usage: Bun runs this with workspace model tests.
// Related: webui/src/pages/workspace/model/live-timeline.ts.
import { expect, test } from 'bun:test';
import { historyRows } from '@/pages/workspace/model/history-rows';
import { liveTimeline } from '@/pages/workspace/model/live-timeline';
import type { Draft } from '@/pages/workspace/model/types';

const pos = (n: number) => `request.${String(n).padStart(20, '0')}`;
const draft: Draft = {
  task: 't',
  agent: 'a',
  generation: '1',
  startSequence: '3',
  presentation: pos(1),
  text: 'Earlier',
  thinking: '',
  truncated: false,
};
const tool = {
  sequence: '10',
  presentation: pos(2),
  event: {
    kind: 'tool_succeeded',
    runId: 'r',
    presentation: pos(2),
    tool: { executionId: pos(2), name: 'execute_command' },
  },
};
test('durable tools stay after an earlier still-active generation', () => {
  const rows = liveTimeline([draft], [], [tool], ['t']);
  expect(rows.map((x) => x.kind)).toEqual(['draft', 'history']);
});
test('persistence replaces the same generation identity and retains its position', () => {
  const saved = {
    sequence: '11',
    presentation: pos(1),
    message: { role: 'assistant', text: 'Earlier' },
  };
  const live = liveTimeline([draft], [], [tool, saved], ['t']);
  const done = liveTimeline([], [], [tool, saved], ['t']);
  expect(live.map((x) => x.key)).toEqual(done.map((x) => x.key));
  expect(done).toHaveLength(2);
  expect(historyRows([tool, saved])[0]).toHaveProperty('item.text', 'Earlier');
});

test('a steering message keeps its position inside the active request', () => {
  const later = {
    sequence: '12',
    presentation: pos(4),
    message: { role: 'assistant', text: 'After steer' },
  };
  const steer = { sequence: '11', presentation: pos(3), message: { role: 'user', text: 'Steer' } };
  const earlier = {
    sequence: '13',
    presentation: pos(1),
    message: { role: 'assistant', text: 'Earlier' },
  };
  expect(
    historyRows([tool, steer, later, earlier]).map((row) =>
      'item' in row ? (row.item.text ?? row.item.event?.tool?.name) : ''
    )
  ).toEqual(['Earlier', 'execute_command', 'Steer', 'After steer']);
});
