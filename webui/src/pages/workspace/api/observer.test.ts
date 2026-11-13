// Purpose: Verify multiplexed observers recover once and never deliver stale socket frames.
// Pattern: Deterministic browser transport and timer contracts.
// Usage: bun test src/pages/workspace/api/observer.test.ts.
// Related: webui/src/pages/workspace/api/observer.ts, webui/tests/event-stream.ts.
import { afterEach, expect, test } from 'bun:test';
import { settleStream, streamFetch, TestSocket } from '#test/event-stream';
import { subscribeEventStream } from '@/pages/workspace/api/observer';

const original = {
  socket: globalThis.WebSocket,
  fetch: globalThis.fetch,
  timer: globalThis.setTimeout,
  clear: globalThis.clearTimeout,
};
let stops: (() => void)[] = [];
afterEach(() => {
  for (const stop of stops) stop();
  stops = [];
  globalThis.WebSocket = original.socket;
  globalThis.fetch = original.fetch;
  globalThis.setTimeout = original.timer;
  globalThis.clearTimeout = original.clear;
});
function fixture() {
  TestSocket.all = [];
  TestSocket.ready = () => {};
  globalThis.WebSocket = TestSocket as unknown as typeof WebSocket;
  globalThis.fetch = streamFetch(async () => Response.json({}));
  const pending = new Map<number, () => void>();
  let id = 0;
  globalThis.setTimeout = ((callback: () => void) => {
    pending.set(++id, callback);
    return id;
  }) as unknown as typeof setTimeout;
  globalThis.clearTimeout = ((id: number) => pending.delete(id)) as unknown as typeof clearTimeout;
  return {
    pending,
    next: async () => {
      const entry = pending.entries().next().value;
      if (!entry) throw new Error('missing retry');
      pending.delete(entry[0]);
      entry[1]();
      await settleStream();
    },
  };
}
test('task and file observers share one connection and release it after the last owner', async () => {
  fixture();
  const subscribe = () =>
    subscribeEventStream({
      connect: async () => ({ url: '/events', events: {} }),
      status: () => {},
      failed: () => true,
    });
  stops.push(subscribe(), subscribe());
  await settleStream();
  expect(TestSocket.all).toHaveLength(1);
  expect(TestSocket.all[0]?.sent).toHaveLength(2);
  stops[0]?.();
  expect(TestSocket.all[0]?.closed).toBe(false);
  stops[1]?.();
  expect(TestSocket.all[0]?.closed).toBe(true);
});
test('failed reconnect preparation schedules a fresh attempt', async () => {
  const clock = fixture();
  let attempts = 0;
  stops.push(
    subscribeEventStream({
      connect: async () => {
        if (++attempts === 2) throw new Error('temporary failure');
        return { url: '/events', events: {} };
      },
      status: () => {},
      failed: () => true,
    })
  );
  await settleStream();
  TestSocket.all[0]?.onerror?.();
  expect(TestSocket.all[0]?.closed).toBe(true);
  await clock.next();
  expect(attempts).toBe(2);
  expect(clock.pending.size).toBe(1);
  await clock.next();
  expect(attempts).toBe(3);
});
test('malformed frames reconnect and a closed source cannot deliver another event', async () => {
  const clock = fixture();
  let delivered = 0;
  stops.push(
    subscribeEventStream({
      connect: async () => ({ url: '/events', events: { event: () => delivered++ } }),
      status: () => {},
      failed: () => true,
    })
  );
  await settleStream();
  const old = TestSocket.all[0]!;
  old.onmessage?.(new MessageEvent('message', { data: '{' }));
  await clock.next();
  old.dispatchEvent(new MessageEvent('event', { data: '{}' }));
  TestSocket.all[1]?.dispatchEvent(new MessageEvent('event', { data: '{}' }));
  expect(delivered).toBe(1);
});
