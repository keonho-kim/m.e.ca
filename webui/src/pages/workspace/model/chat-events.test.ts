// Purpose: Prevent duplicate tool lifecycle rows and responses rendered as system metadata.
// Pattern: Pure chat projection regression tests.
// Usage: bun test src/pages/workspace/model/chat-events.test.ts.
// Related: webui/src/pages/workspace/model/chat-events.ts.
import { expect, test } from 'bun:test';
import { liveActivity } from '@/pages/workspace/model/chat-events';
import type { EventJson } from '@/shared/api';

const started: EventJson = {
  runId: 'r',
  agent: 'a',
  kind: 'tool_started',
  tool: { callId: '1', name: 'execute_command' },
};
const result: EventJson = { ...started, kind: 'tool_succeeded', text: 'done' };

test('a saved command result takes over from all live states of that call', () => {
  expect(liveActivity([started, result], [{ event: result }])).toEqual([]);
  expect(liveActivity([started, result], [])).toEqual([result]);
});
test('same-named tools from other runs, agents and calls remain independent', () => {
  const others = [
    { ...result, agent: 'b' },
    { ...result, runId: 'other' },
    { ...result, tool: { ...result.tool, callId: '2' } },
  ];
  expect(liveActivity([result, ...others], [{ event: result }])).toEqual(others);
});
test('response text never returns to the metadata activity area', () => {
  expect(
    liveActivity(
      ['progress', 'team_progress', 'team_message', 'team_discussion'].map((kind) => ({
        kind,
        text: 'answer',
      })),
      []
    )
  ).toEqual([]);
});
