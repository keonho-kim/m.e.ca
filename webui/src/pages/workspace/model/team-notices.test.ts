// Purpose: Keep internal team join/leave lifecycle frames out of the visible conversation transcript.
// Pattern: Native history presentation boundary regression.
// Usage: bun test src/pages/workspace/model/team-notices.test.ts.
// Related: webui/src/pages/workspace/model/history-rows.ts, webui/src/pages/workspace/ui/chat/live-activity.tsx.
import { expect, test } from 'bun:test';
import { historyRows } from '@/pages/workspace/model/history-rows';

test('team lifecycle remains control data while the answer and aggregate measurements remain visible', () => {
  const rows = historyRows([
    { sequence: '1', event: { kind: 'team_notice', agent: 'Hana' } },
    { sequence: '2', event: { kind: 'team_notice', agent: 'Ren' } },
    {
      sequence: '3',
      event: {
        kind: 'team_progress',
        agent: 'Hana',
        message: { role: 'assistant', text: 'Actual answer' },
      },
    },
    { sequence: '4', event: { kind: 'team_notice', agent: 'Ren' } },
    { sequence: '5', completion: { status: 'COMPLETED', statistics: { elapsedMillis: '1000' } } },
  ]);
  expect(rows).toHaveLength(2);
  expect('item' in rows[0]! && rows[0].item.text).toBe('Actual answer');
  expect('completion' in rows[1]!).toBe(true);
});
