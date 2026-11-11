// Purpose: Preserve one assistant presentation across reasoning persistence and tool calls.
// Pattern: History grouping regression tests.
// Usage: bun test src/pages/workspace/model/history-rows.test.ts.
// Related: webui/src/pages/workspace/model/history-rows.ts.
import { expect, test } from 'bun:test';
import { historyRows } from '@/pages/workspace/model/history-rows';

test('reasoning and adjacent assistant text form one bubble without swallowing tools or outcomes', () => {
  const thought = { sequence: '1', event: { kind: 'thinking', thinking: 'Reason' } };
  const answer = { sequence: '2', role: 'assistant', text: 'Answer' };
  const tool = { sequence: '3', event: { kind: 'tool_succeeded', tool: { name: 'read_file' } } };
  const outcome = { sequence: '4', outcome: { status: 'completed' } };
  expect(historyRows([thought, answer, tool, outcome])).toEqual([
    { item: answer, thinking: thought },
    { item: tool },
    { item: outcome },
  ]);
});
test('different agents and non-adjacent thoughts never merge', () => {
  expect(
    historyRows([
      { event: { thinking: 'private', agent: 'peer' } },
      { role: 'assistant', text: 'other' },
    ])
  ).toHaveLength(2);
});

test('one saved tool result replaces its started and proposed records', () => {
  const tool = { callId: 'command', name: 'execute_command' };
  const event = { runId: 'run', agent: 'peer', tool };
  const finished = { sequence: '3', event: { ...event, kind: 'tool_succeeded' } };
  expect(
    historyRows([
      { sequence: '1', event: { ...event, kind: 'tool_proposed' } },
      { sequence: '2', event: { ...event, kind: 'tool_started' } },
      finished,
    ])
  ).toEqual([{ item: finished }]);
});

test('public team responses are messages rather than system metadata', () => {
  const item = {
    sequence: '9',
    event: { kind: 'team_progress', agent: 'peer', text: 'My answer' },
  };
  expect(historyRows([item])).toEqual([
    { item: { ...item, role: 'assistant', text: 'My answer' } },
  ]);
});
