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
  tool: { callId: '1', executionId: 'execution-1', name: 'execute_command' },
};
const result: EventJson = {
  ...started,
  kind: 'tool_succeeded',
  tool: { ...started.tool, summary: 'done' },
};

test('a saved command result takes over from all live states of that call', () => {
  expect(liveActivity([started, result], [{ event: result }])).toEqual([]);
  expect(liveActivity([started, result], [])).toEqual([result]);
});
test('same-named tools from other runs, agents and calls remain independent', () => {
  const others = [
    { ...result, agent: 'b' },
    { ...result, runId: 'other' },
    { ...result, tool: { ...result.tool, callId: '2', executionId: 'execution-2' } },
  ];
  expect(liveActivity([result, ...others], [{ event: result }])).toEqual(others);
});

test('reused provider call IDs never hide a later independent execution', () => {
  const next = { ...started, tool: { ...started.tool, executionId: 'execution-2' } };
  expect(liveActivity([next], [{ event: result }])).toEqual([next]);
});
test('streamed progress stays with its draft and direct team speech retains a message body', () => {
  const events = ['progress', 'team_progress', 'team_message', 'team_discussion'].map((kind) => ({
    kind,
    message: { role: 'assistant', text: 'answer' },
  }));
  const visible = liveActivity(events, []);
  expect(visible.map((event) => event.kind)).toEqual(['team_message', 'team_discussion']);
  expect(visible.every((event) => event.message?.text === 'answer' && !event.notice)).toBe(true);
});

test('original tool references deduplicate rejected calls without an execution identity', () => {
  const rejected = {
    kind: 'tool_failed',
    tool: { name: 'unknown', reference: { sessionId: 's', sequence: '3' } },
  };
  expect(liveActivity([rejected], [{ sequence: '5', event: rejected }])).toEqual([]);
});

test('stored steer removes its live echo without hiding an identical later instruction', () => {
  const first = {
    kind: 'user_message',
    presentation: 'scope.00000000000000000001',
    message: { role: 'user', text: 'Again' },
  };
  const second = { ...first, presentation: 'scope.00000000000000000002' };
  expect(
    liveActivity(
      [first, second],
      [{ sequence: '8', presentation: first.presentation, message: first.message }]
    )
  ).toEqual([second]);
});
