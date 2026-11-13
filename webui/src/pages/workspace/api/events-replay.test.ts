// Purpose: Keep replayed control frames from reversing query-cache task state after completion.
// Pattern: Deterministic event-observer boundary regression.
// Usage: bun test src/pages/workspace/api/events-replay.test.ts.
// Related: webui/src/pages/workspace/api/events.ts, webui/src/pages/workspace/model/history-handoff.ts.
import { expect, test } from 'bun:test';
import { TestSocket as Connection, streamFetch } from '#test/event-stream';
import { observe } from '@/pages/workspace/api/events';
import { useWorkspaceStore } from '@/pages/workspace/model/store';
import type { EventJson, SnapshotJson } from '@/shared/api';
import { queries } from '@/shared/api';

test('duplicate and old-epoch controls cannot overwrite a completed task in the query cache', async () => {
  const original = {
    fetch: globalThis.fetch,
    WebSocket: globalThis.WebSocket,
    state: useWorkspaceStore.getState(),
  };
  const target = { workspace: 'w', session: 's' };
  const ready = new Promise<Connection>((resolve) => {
    Connection.ready = resolve;
  });

  let dispose: (() => void) | undefined;
  try {
    queries.clear();
    useWorkspaceStore.setState({ ...useWorkspaceStore.getInitialState(), selection: target });
    globalThis.WebSocket = Connection as unknown as typeof WebSocket;
    globalThis.fetch = streamFetch(async () =>
      Response.json({ state: { cursor: 'epoch:1', workspaceId: 'w' } })
    );
    dispose = observe(target);
    const source = await ready;
    const emit = (event: EventJson) =>
      source.dispatchEvent(new MessageEvent('event', { data: JSON.stringify({ event }) }));
    const identity = { workspaceId: 'w', sessionId: 's', taskId: 't' };
    emit({ ...identity, kind: 'request_finished', cursor: 'epoch:3', status: 'answered' });
    for (const cursor of ['epoch:2', 'epoch:3', 'old:99']) {
      emit({ ...identity, kind: 'request_started', cursor, status: 'running' });
      expect(queries.getQueryData<SnapshotJson>(['state', 'w', 's'])?.task?.status).toBe(
        'answered'
      );
    }
    expect(useWorkspaceStore.getState().tasks.t?.status).toBe('answered');
  } finally {
    dispose?.();
    queries.clear();
    globalThis.fetch = original.fetch;
    globalThis.WebSocket = original.WebSocket;
    useWorkspaceStore.setState(original.state);
  }
});
