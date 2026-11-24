// Purpose: Verify durable intermediate responses hand off before subsequent tools and output.
// Pattern: Stream-to-history ownership regression tests.
// Usage: bun test src/pages/workspace/model/stream-handoff.test.ts.
// Related: webui/src/pages/workspace/model/stream-projection.ts, webui/src/pages/workspace/model/stream-slice.ts.
import { expect, test } from 'bun:test';
import { useWorkspaceStore } from '@/pages/workspace/model/store';
import { applyEvents, snapshotDrafts } from '@/pages/workspace/model/stream-projection';

test('committed progress settles only its agent draft without adding a system response', () => {
  const base = { workspaceId: 'w', sessionId: 's', taskId: 't' };
  const initial = {
    ...useWorkspaceStore.getInitialState(),
    selection: { workspace: 'w', session: 's' },
  };
  const streamed = {
    ...initial,
    ...applyEvents(initial, [
      {
        ...base,
        kind: 'assistant_delta',
        agent: 'a',
        message: { role: 'assistant', text: 'Before command' },
      },
      {
        ...base,
        kind: 'assistant_delta',
        agent: 'b',
        message: { role: 'assistant', text: 'Another agent' },
      },
    ]),
  };
  const result = applyEvents(streamed, [
    {
      ...base,
      kind: 'team_progress',
      agent: 'a',
      message: { role: 'assistant', text: 'Before command' },
    },
  ]);
  expect(result.drafts?.['t/a']?.doneRevision).toBe(1);
  expect(result.drafts?.['t/b']?.doneRevision).toBeUndefined();
  expect(result.events?.some((event) => event.kind === 'team_progress')).toBe(false);
});

test('reconnect keeps committed output until fresh history confirms it', () => {
  const previous = useWorkspaceStore.getState();
  const target = { workspace: 'w', session: 's' };
  try {
    useWorkspaceStore.setState({ ...useWorkspaceStore.getInitialState(), selection: target });
    useWorkspaceStore.getState().bootstrap(target, {
      drafts: [
        { taskId: 't', agent: 'a', text: 'Saved', committed: true },
        { taskId: 't', agent: 'b', text: 'Streaming' },
      ],
    });
    const revision = useWorkspaceStore.getState().historyRevision;
    useWorkspaceStore.getState().acknowledgeHistory(target, revision - 1, { through: '0' });
    expect(useWorkspaceStore.getState().drafts['t/a']?.text).toBe('Saved');
    useWorkspaceStore.getState().acknowledgeHistory(target, revision, { through: '0' });
    expect(useWorkspaceStore.getState().drafts['t/a']).toBeUndefined();
    expect(useWorkspaceStore.getState().drafts['t/b']?.text).toBe('Streaming');
  } finally {
    useWorkspaceStore.setState(previous);
  }
});

test('completed team reply replaces its streamed prefix before history arrives', () => {
  const base = { workspaceId: 'w', sessionId: 's', taskId: 't', agent: 'Ren' };
  const initial = {
    ...useWorkspaceStore.getInitialState(),
    selection: { workspace: 'w', session: 's' },
  };
  const streamed = {
    ...initial,
    ...applyEvents(initial, [
      {
        ...base,
        kind: 'assistant_delta',
        presentation: 'reply.1',
        generation: '2',
        message: { text: 'Partial reply' },
      },
    ]),
  };
  const completed = applyEvents(streamed, [
    {
      ...base,
      kind: 'team_progress',
      presentation: 'reply.1',
      historySequence: '12',
      message: { text: 'Completed revised answer' },
    },
  ]);
  expect(completed.drafts?.['t/Ren']?.text).toBe('Completed revised answer');
  expect(completed.drafts?.['t/Ren']?.doneThrough).toBe('12');
  expect(streamed.drafts['t/Ren']?.text).toBe('Partial reply');
});

test('nonstreamed team reply is visible while durable history is delayed', () => {
  const initial = {
    ...useWorkspaceStore.getInitialState(),
    selection: { workspace: 'w', session: 's' },
  };
  const completed = applyEvents(initial, [
    {
      workspaceId: 'w',
      sessionId: 's',
      taskId: 't',
      agent: 'Ren',
      kind: 'team_progress',
      presentation: 'reply.1',
      historySequence: '12',
      message: { text: 'Final answer without deltas' },
    },
  ]);
  expect(completed.drafts?.['t/Ren']?.text).toBe('Final answer without deltas');
  expect(completed.drafts?.['t/Ren']?.doneThrough).toBe('12');
});

test('reconnect uses the matching committed preview after the server releases draft memory', () => {
  const drafts = snapshotDrafts(
    {
      drafts: [
        {
          taskId: 't',
          agent: 'Ren',
          presentation: 'reply.1',
          text: '',
          committed: true,
          truncated: true,
          historySequence: '12',
        },
      ],
      activity: [
        {
          taskId: 't',
          agent: 'Hana',
          kind: 'team_progress',
          presentation: 'other.1',
          message: { text: 'Other peer' },
        },
        {
          taskId: 't',
          agent: 'Ren',
          kind: 'team_progress',
          presentation: 'reply.1',
          historySequence: '12',
          message: { text: 'Saved response preview' },
        },
      ],
    },
    3
  );
  expect(drafts['t/Ren']?.text).toBe('Saved response preview');
  expect(drafts['t/Ren']?.doneThrough).toBe('12');
});

test('same-server reconnect preserves completed output until history confirms it', () => {
  const previous = useWorkspaceStore.getState();
  const target = { workspace: 'w', session: 's' };
  try {
    useWorkspaceStore.setState({
      ...useWorkspaceStore.getInitialState(),
      selection: target,
      replayCursor: 'epoch:2',
    });
    useWorkspaceStore.getState().receive([
      {
        workspaceId: 'w',
        sessionId: 's',
        taskId: 't',
        kind: 'request_started',
        status: 'running',
        cursor: 'epoch:3',
      },
      {
        workspaceId: 'w',
        sessionId: 's',
        taskId: 't',
        kind: 'progress',
        presentation: 'reply.1',
        historySequence: '4',
        message: { text: 'Saved contribution' },
        cursor: 'epoch:4',
      },
    ]);
    const snapshot = { cursor: 'epoch:5', tasks: [{ id: 't', sessionId: 's', status: 'running' }] };
    useWorkspaceStore.getState().bootstrap(target, snapshot);
    expect(Object.values(useWorkspaceStore.getState().drafts).map((d) => d.text)).toContain(
      'Saved contribution'
    );
    const revision = useWorkspaceStore.getState().historyRevision;
    useWorkspaceStore.getState().acknowledgeHistory(target, revision, {
      through: '4',
      items: [
        {
          sequence: '4',
          presentation: 'reply.1',
          message: { role: 'assistant', text: 'Saved contribution' },
        },
      ],
    });
    expect(useWorkspaceStore.getState().drafts).toEqual({});
    useWorkspaceStore.getState().receive([
      {
        workspaceId: 'w',
        sessionId: 's',
        taskId: 't',
        kind: 'progress',
        presentation: 'reply.2',
        message: { text: 'Old server output' },
        cursor: 'epoch:6',
      },
    ]);
    useWorkspaceStore.getState().bootstrap(target, { ...snapshot, cursor: 'new:1' });
    expect(useWorkspaceStore.getState().drafts).toEqual({});
  } finally {
    useWorkspaceStore.setState(previous);
  }
});

test('a later snapshot generation preserves the preceding decision until history arrives', () => {
  const previous = {
    't/': {
      task: 't',
      agent: '',
      generation: '1',
      presentation: 'reply.1',
      startSequence: '3',
      text: '',
      thinking: 'Earlier reasoning',
      truncated: false,
    },
  };
  const drafts = snapshotDrafts(
    {
      tasks: [{ id: 't', status: 'running' }],
      drafts: [
        {
          taskId: 't',
          generation: '2',
          presentation: 'reply.2',
          startSequence: '7',
          thinking: 'Current reasoning',
        },
      ],
    },
    4,
    previous
  );
  expect(drafts['t//1']?.thinking).toBe('Earlier reasoning');
  expect(drafts['t/']?.thinking).toBe('Current reasoning');
  expect(previous['t/'].generation).toBe('1');
});
