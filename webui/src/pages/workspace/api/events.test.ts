// Purpose: Verify that application silence does not close a healthy native SSE connection.
// Pattern: Deterministic transport lifecycle regression test.
// Usage: bun test src/pages/workspace/api/events.test.ts with a fake WebSocket and clock.
// Related: webui/src/pages/workspace/api/events.ts, internal/serverapi/eventstream/writer.go.
import { expect, test } from 'bun:test';
import { TestSocket as Connection, streamFetch } from '#test/event-stream';
import { observe } from '@/pages/workspace/api/events';
import { clearFollowups } from '@/pages/workspace/model/followups';
import { waitSelections } from '@/pages/workspace/model/reference-queue';
import { useWorkspaceStore } from '@/pages/workspace/model/store';
import { awaitSubmission, clearPending } from '@/pages/workspace/model/submission';
import { queries } from '@/shared/api';

test('an idle observer remains open without application updates', async () => {
  const original = {
    fetch: globalThis.fetch,
    WebSocket: globalThis.WebSocket,
    setInterval: globalThis.setInterval,
    clearInterval: globalThis.clearInterval,
    now: Date.now,
    state: useWorkspaceStore.getState(),
  };
  const ticks: Array<() => void> = [];
  let now = 1000;
  const ready = new Promise<Connection>((resolve) => {
    Connection.ready = resolve;
  });

  let cleanup: (() => void) | undefined;
  try {
    useWorkspaceStore.setState({ queues: {}, tasks: {} });
    globalThis.WebSocket = Connection as unknown as typeof WebSocket;
    globalThis.fetch = streamFetch(async () =>
      Response.json({ state: { cursor: 'epoch:0', workspaceId: 'w', tasks: [] } })
    );
    globalThis.setInterval = ((callback: () => void) => {
      ticks.push(callback);
      return 1;
    }) as unknown as typeof setInterval;
    globalThis.clearInterval = () => {};
    Date.now = () => now;
    cleanup = observe({ workspace: 'w', session: 's' });
    const source = await ready;
    now += 120000;
    for (const tick of ticks) tick();
    expect(source.closed).toBe(false);
    expect(useWorkspaceStore.getState().streamStatus).toBe('connected');
    cleanup();
    cleanup = undefined;
    expect(source.closed).toBe(true);
  } finally {
    cleanup?.();
    Date.now = original.now;
    globalThis.fetch = original.fetch;
    globalThis.WebSocket = original.WebSocket;
    globalThis.setInterval = original.setInterval;
    globalThis.clearInterval = original.clearInterval;
    useWorkspaceStore.setState(original.state);
  }
});

test('reconnecting reconciles a queued predecessor already made terminal by an interrupted snapshot', async () => {
  const original = {
    fetch: globalThis.fetch,
    WebSocket: globalThis.WebSocket,
    state: useWorkspaceStore.getState(),
  };
  const target = { workspace: 'w', session: 'queued' };
  let started!: () => void;
  const ready = new Promise<void>((resolve) => {
    started = resolve;
  });
  const starts: string[] = [];

  let cleanup: (() => void) | undefined;
  try {
    queries.clear();
    clearFollowups();
    clearPending();
    const historyKey = ['history', target.workspace, target.session, '0'];
    queries.setQueryData(historyKey, { items: [] });
    const usageKey = ['session-usage', target.workspace, target.session];
    queries.setQueryData(usageKey, { input: '0', output: '0' });
    useWorkspaceStore.setState({
      queues: {},
      pausedQueues: {},
      tasks: { previous: { id: 'previous', ...target, status: 'answered' } },
    });
    useWorkspaceStore
      .getState()
      .enqueue(target, { text: 'captured', paths: [], files: [], revision: 0 });
    globalThis.WebSocket = Connection as unknown as typeof WebSocket;
    globalThis.fetch = streamFetch(async (_url: string | URL | Request, options?: RequestInit) => {
      const command = JSON.parse(String(options?.body));
      if (command.state)
        return Response.json({
          state: {
            workspaceId: 'w',
            cursor: 'epoch:1',
            task: { id: 'previous', sessionId: target.session, status: 'answered' },
          },
        });
      if (command.taskStatus)
        return Response.json({
          task: {
            id: 'previous',
            sessionId: target.session,
            status: 'answered',
            outcome: 'completed',
          },
        });
      if (command.start) {
        starts.push(command.start.text);
        started();
        return Response.json({ task: { id: 'next', status: 'running' } });
      }
      throw new Error('Unexpected fixture command');
    });
    cleanup = observe(target);
    await ready;
    await awaitSubmission(target);
    expect(starts).toEqual(['captured']);
    expect(queries.getQueryState(historyKey)?.isInvalidated).toBe(true);
    expect(queries.getQueryState(usageKey)?.isInvalidated).toBe(true);
  } finally {
    cleanup?.();
    clearFollowups();
    clearPending();
    queries.clear();
    globalThis.fetch = original.fetch;
    globalThis.WebSocket = original.WebSocket;
    useWorkspaceStore.setState(original.state);
  }
});

test('an authoritative snapshot replaces pending deltas and releases its cached live body', async () => {
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
    globalThis.fetch = streamFetch(async (_url: string | URL | Request, options?: RequestInit) =>
      Response.json(
        JSON.parse(String(options?.body)).fileSelections
          ? { fileSelections: { paths: [] } }
          : { state: { cursor: 'epoch:1', workspaceId: 'w' } }
      )
    );
    dispose = observe(target);
    const source = await ready;
    const emit = (name: string, data: object) =>
      source.dispatchEvent(new MessageEvent(name, { data: JSON.stringify(data) }));
    const identity = {
      workspaceId: 'w',
      sessionId: 's',
      taskId: 't',
      agent: 'main',
      generation: '1',
    };
    emit('event', {
      event: {
        ...identity,
        kind: 'assistant_delta',
        message: { role: 'assistant', text: 'first' },
      },
    });
    emit('snapshot', {
      snapshot: {
        cursor: 'epoch:2',
        workspaceId: 'w',
        drafts: [{ ...identity, text: 'first' }],
        activity: [],
      },
    });
    emit('event', {
      event: {
        ...identity,
        kind: 'assistant_delta',
        message: { role: 'assistant', text: ' second' },
      },
    });
    emit('event', {
      event: {
        ...identity,
        kind: 'progress',
        message: { role: 'assistant', text: 'first second' },
      },
    });
    expect(useWorkspaceStore.getState().drafts['t/main']?.text).toBe('first second');
    expect(queries.getQueryData(['state', 'w', 's'])).toEqual({
      cursor: 'epoch:2',
      workspaceId: 'w',
    });
    await waitSelections(target);
  } finally {
    dispose?.();
    queries.clear();
    globalThis.fetch = original.fetch;
    globalThis.WebSocket = original.WebSocket;
    useWorkspaceStore.setState(original.state);
  }
});
