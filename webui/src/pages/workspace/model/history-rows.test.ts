// Purpose: Preserve one assistant presentation across reasoning persistence and tool calls.
// Pattern: History grouping regression tests.
// Usage: bun test src/pages/workspace/model/history-rows.test.ts.
// Related: webui/src/pages/workspace/model/history-rows.ts.
import { expect, test } from 'bun:test';
import { historyRows } from '@/pages/workspace/model/history-rows';
import type { EventJson } from '@/shared/api';

test('code execution keeps inner calls under their parent while parent status changes', () => {
  const parent = {
    runId: 'run',
    kind: 'tool_started',
    tool: { executionId: 'code', name: 'codemode' },
  };
  const child = {
    runId: 'run',
    kind: 'tool_succeeded',
    tool: { executionId: 'child', parentExecutionId: 'code', name: 'write_file' },
  };
  const finished = { ...parent, kind: 'tool_succeeded' };
  const rows = historyRows([{ event: parent }, { event: child }, { event: finished }]);
  expect(rows).toHaveLength(1);
  const row = rows[0];
  expect(row && 'item' in row && row.tools?.map((item) => item.event)).toEqual([finished, child]);
  expect(historyRows([{ event: parent }, { event: { ...child, agent: 'peer' } }])).toHaveLength(2);
});

test('reasoning and adjacent assistant text form one bubble without swallowing tools or outcomes', () => {
  const thought = {
    sequence: '1',
    event: { kind: 'thinking', message: { role: 'assistant', thinking: 'Reason' } },
  };
  const answer = { sequence: '2', message: { role: 'assistant', text: 'Answer' } };
  const tool = {
    sequence: '3',
    event: { kind: 'tool_succeeded', tool: { name: 'execute_command' } },
  };
  const completion = { sequence: '4', completion: { status: 'COMPLETED' as const } };
  const rows = historyRows([thought, answer, tool, completion]);
  expect(rows).toHaveLength(3);
  const first = rows[0];
  expect(first && 'item' in first && first.item.text).toBe('Answer');
  expect(first && 'item' in first && first.thinking?.thinking).toBe('Reason');
  expect(rows[1] && 'item' in rows[1] && rows[1].item.event?.tool?.name).toBe('execute_command');
  expect(rows[2]).toEqual({ sequence: '4', completion: completion.completion });
});

test('different agents and non-adjacent thoughts never merge', () => {
  expect(
    historyRows([
      {
        event: {
          agent: 'peer',
          kind: 'thinking',
          message: { role: 'assistant', thinking: 'private' },
        },
      },
      { message: { role: 'assistant', text: 'other' } },
    ])
  ).toHaveLength(2);
});

test('one saved tool result replaces its started and proposed records', () => {
  const tool = { callId: 'command', executionId: 'execution-command', name: 'execute_command' };
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
    event: {
      kind: 'team_progress',
      agent: 'peer',
      message: { role: 'assistant', text: 'My answer' },
    },
  };
  expect(historyRows([item])).toEqual([
    {
      item: {
        sequence: '9',
        event: item.event,
        role: 'assistant',
        text: 'My answer',
        truncated: undefined,
      },
    },
  ]);
});

test('reused provider call IDs retain distinct saved executions', () => {
  const tool = { callId: 'reused', name: 'execute_command' };
  const first = {
    sequence: '1',
    event: { runId: 'run', kind: 'tool_succeeded', tool: { ...tool, executionId: 'one' } },
  };
  const second = {
    sequence: '2',
    event: { runId: 'run', kind: 'tool_succeeded', tool: { ...tool, executionId: 'two' } },
  };
  expect(historyRows([first, second])).toEqual([{ item: first }, { item: second }]);
});

const batchEvent = (executionId: string, batchId = 'wave', agent = 'peer'): EventJson => ({
  kind: 'tool_started',
  runId: 'run',
  agent,
  tool: { executionId, batchId, name: 'execute_command' },
});

test('parallel lifecycle updates replace their own entry beneath one agent heading', () => {
  const first = batchEvent('one'),
    second = batchEvent('two');
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
    batchEvent('1'),
    batchEvent('2', 'wave', 'other'),
    batchEvent('3', 'next'),
    batchEvent('4', ''),
    { ...batchEvent('5'), runId: 'next-run' },
  ];
  expect(historyRows(entries.map((event) => ({ event })))).toHaveLength(5);
});
