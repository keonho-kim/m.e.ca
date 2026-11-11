// Purpose: Verify closed or malformed SSE connections always recover without stale delivery.
// Pattern: Deterministic browser transport and timer fixtures.
// Usage: bun test src/shared/api/event-stream.test.ts.
// Related: webui/src/shared/api/event-stream.ts.
import { afterEach, expect, test } from 'bun:test';
import { subscribeEventStream } from '@/shared/api/event-stream';

const original = {
  source: globalThis.EventSource,
  timer: globalThis.setTimeout,
  clear: globalThis.clearTimeout,
};
let stop: (() => void) | undefined;
afterEach(() => {
  stop?.();
  stop = undefined;
  globalThis.EventSource = original.source;
  globalThis.setTimeout = original.timer;
  globalThis.clearTimeout = original.clear;
});
const settle = async () => {
  for (let n = 0; n < 8; n++) await Promise.resolve();
};
class Source {
  static all: Source[] = [];
  onopen: (() => void) | null = null;
  onerror: (() => void) | null = null;
  closed = false;
  handlers: Record<string, (event: { data: string }) => void> = {};
  constructor() {
    Source.all.push(this);
  }
  close() {
    this.closed = true;
  }
  addEventListener(name: string, handle: (event: { data: string }) => void) {
    this.handlers[name] = handle;
  }
}
function fixture() {
  Source.all = [];
  globalThis.EventSource = Source as unknown as typeof EventSource;
  const pending = new Map<number, () => void>();
  let id = 0;
  globalThis.setTimeout = ((callback: () => void) => {
    pending.set(++id, callback);
    return id;
  }) as unknown as typeof setTimeout;
  globalThis.clearTimeout = ((id: number) => pending.delete(id)) as unknown as typeof clearTimeout;
  return {
    next: async () => {
      const entry = pending.entries().next().value;
      if (!entry) throw new Error('missing reconnect');
      pending.delete(entry[0]);
      entry[1]();
      await settle();
    },
    pending,
  };
}
test('a failed snapshot during reconnection schedules another attempt', async () => {
  const clock = fixture();
  let attempts = 0;
  stop = subscribeEventStream({
    connect: async () => {
      if (++attempts === 2) throw new Error('temporary read failure');
      return { url: '/events', events: {} };
    },
    status: () => {},
    failed: () => true,
  });
  await settle();
  Source.all[0]?.onerror?.();
  expect(Source.all[0]?.closed).toBe(true);
  await clock.next();
  expect(attempts).toBe(2);
  expect(clock.pending.size).toBe(1);
  await clock.next();
  expect(Source.all).toHaveLength(2);
  stop();
  expect(clock.pending.size).toBe(0);
});
test('malformed events reconnect and old source callbacks cannot deliver again', async () => {
  const clock = fixture();
  let delivered = 0;
  stop = subscribeEventStream({
    connect: async () => ({
      url: '/events',
      events: {
        event: (data) => {
          JSON.parse(data);
          delivered++;
        },
      },
    }),
    status: () => {},
    failed: () => true,
  });
  await settle();
  const old = Source.all[0];
  old?.handlers.event?.({ data: '{' });
  await clock.next();
  old?.handlers.event?.({ data: '{}' });
  Source.all[1]?.handlers.event?.({ data: '{}' });
  expect(delivered).toBe(1);
});
