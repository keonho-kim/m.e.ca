// Purpose: Verify durable intermediate responses hand off before subsequent tools and output.
// Pattern: Stream-to-history ownership regression tests.
// Usage: bun test src/pages/workspace/model/stream-handoff.test.ts.
// Related: webui/src/pages/workspace/model/stream-projection.ts, webui/src/pages/workspace/model/stream-slice.ts.
import { expect, test } from 'bun:test';
import { useWorkspaceStore } from '@/pages/workspace/model/store';
import { applyEvents } from '@/pages/workspace/model/stream-projection';

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
