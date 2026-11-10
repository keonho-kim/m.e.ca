// Purpose: Verify that application silence does not close a healthy native SSE connection.
// Pattern: Deterministic transport lifecycle regression test.
// Usage: bun test src/pages/workspace/api/events.test.ts with a fake EventSource and clock.
// Related: webui/src/pages/workspace/api/events.ts, internal/serverapi/eventstream/writer.go.
import { expect, test } from 'bun:test';
import { observe } from '@/pages/workspace/api/events';
import { clearFollowups } from '@/pages/workspace/model/followups';
import { useWorkspaceStore } from '@/pages/workspace/model/store';
import { awaitSubmission, clearPending } from '@/pages/workspace/model/submission';
import { queries } from '@/shared/api';

test('an idle EventSource remains open when comment heartbeats produce no JavaScript message', async () => {
  const original = {
    fetch: globalThis.fetch,
    EventSource: globalThis.EventSource,
    setInterval: globalThis.setInterval,
    clearInterval: globalThis.clearInterval,
    now: Date.now,
    state: useWorkspaceStore.getState(),
  };
  const ticks: Array<() => void> = [];
  let now = 1000;
  let created!: (value: Connection) => void;
  const ready = new Promise<Connection>((resolve) => {
    created = resolve;
  });
  class Connection {
    static CLOSED = 2;
    readyState = 1;
    closed = false;
    onopen: (() => void) | null = null;
    onerror: (() => void) | null = null;
    constructor() {
      created(this);
    }
    addEventListener() {}
    close() {
      this.closed = true;
      this.readyState = 2;
    }
  }
  let cleanup: (() => void) | undefined;
  try {
    useWorkspaceStore.setState({ queues: {}, tasks: {} });
    globalThis.EventSource = Connection as unknown as typeof EventSource;
    globalThis.fetch = Object.assign(
      async () => Response.json({ state: { cursor: 'idle-cursor', tasks: [] } }),
      { preconnect: original.fetch.preconnect }
    );
    globalThis.setInterval = ((callback: () => void) => {
      ticks.push(callback);
      return 1;
    }) as unknown as typeof setInterval;
    globalThis.clearInterval = () => {};
    Date.now = () => now;
    cleanup = observe({ workspace: 'w', session: 's' });
    const source = await ready;
    source.onopen?.();
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
    globalThis.EventSource = original.EventSource;
    globalThis.setInterval = original.setInterval;
    globalThis.clearInterval = original.clearInterval;
    useWorkspaceStore.setState(original.state);
  }
});

test('reconnecting reconciles a queued predecessor already made terminal by an interrupted snapshot', async () => {
  const original = {
    fetch: globalThis.fetch,
    EventSource: globalThis.EventSource,
    state: useWorkspaceStore.getState(),
  };
  const target = { workspace: 'w', session: 'queued' };
  let started!: () => void;
  const ready = new Promise<void>((resolve) => {
    started = resolve;
  });
  const starts: string[] = [];
  class Connection {
    static CLOSED = 2;
    readyState = 1;
    addEventListener() {}
    close() {
      this.readyState = 2;
    }
  }
  let cleanup: (() => void) | undefined;
  try {
    queries.clear();
    clearFollowups();
    clearPending();
    const historyKey = ['history', target.workspace, target.session, '0'];
    queries.setQueryData(historyKey, { items: [] });
    useWorkspaceStore.setState({
      queues: {},
      pausedQueues: {},
      tasks: { previous: { id: 'previous', ...target, status: 'answered' } },
    });
    useWorkspaceStore
      .getState()
      .enqueue(target, { text: 'captured', paths: [], files: [], revision: 0 });
    globalThis.EventSource = Connection as unknown as typeof EventSource;
    globalThis.fetch = Object.assign(
      async (_url: string | URL | Request, options?: RequestInit) => {
        const command = JSON.parse(String(options?.body));
        if (command.state)
          return Response.json({
            state: {
              cursor: 'after-completion',
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
      },
      { preconnect: original.fetch.preconnect }
    );
    cleanup = observe(target);
    await ready;
    await awaitSubmission(target);
    expect(starts).toEqual(['captured']);
    expect(queries.getQueryState(historyKey)?.isInvalidated).toBe(true);
  } finally {
    cleanup?.();
    clearFollowups();
    clearPending();
    queries.clear();
    globalThis.fetch = original.fetch;
    globalThis.EventSource = original.EventSource;
    useWorkspaceStore.setState(original.state);
  }
});

test('an authoritative snapshot replaces pending deltas and releases its cached live body', async () => {
  const original = {
    fetch: globalThis.fetch,
    EventSource: globalThis.EventSource,
    state: useWorkspaceStore.getState(),
  };
  const target = { workspace: 'w', session: 's' };
  let opened!: (source: Connection) => void;
  const ready = new Promise<Connection>((resolve) => {
    opened = resolve;
  });
  class Connection extends EventTarget {
    static CLOSED = 2;
    readyState = 1;
    constructor() {
      super();
      opened(this);
    }
    close() {
      this.readyState = 2;
    }
  }
  let dispose: (() => void) | undefined;
  try {
    queries.clear();
    useWorkspaceStore.setState({ ...useWorkspaceStore.getInitialState(), selection: target });
    globalThis.EventSource = Connection as unknown as typeof EventSource;
    globalThis.fetch = Object.assign(
      async () => Response.json({ state: { cursor: '1', workspaceId: 'w' } }),
      { preconnect: original.fetch.preconnect }
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
    emit('event', { event: { ...identity, kind: 'assistant_delta', text: 'first' } });
    emit('snapshot', {
      snapshot: {
        cursor: '2',
        workspaceId: 'w',
        drafts: [{ ...identity, text: 'first' }],
        activity: [],
      },
    });
    emit('event', { event: { ...identity, kind: 'assistant_delta', text: ' second' } });
    emit('event', { event: { ...identity, kind: 'progress' } });
    expect(useWorkspaceStore.getState().drafts['t/main']?.text).toBe('first second');
    expect(queries.getQueryData(['state', 'w', 's'])).toEqual({ cursor: '2', workspaceId: 'w' });
  } finally {
    dispose?.();
    queries.clear();
    globalThis.fetch = original.fetch;
    globalThis.EventSource = original.EventSource;
    useWorkspaceStore.setState(original.state);
  }
});
