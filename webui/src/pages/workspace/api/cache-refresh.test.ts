// Purpose: Verify event bursts cause one refresh and disposal releases deferred query references.
// Pattern: Query observer contract with an explicitly controlled clock boundary.
// Usage: bun test src/pages/workspace/api/cache-refresh.test.ts.
// Related: webui/src/pages/workspace/api/cache-refresh.ts.
import { expect, test } from 'bun:test';
import { QueryObserver } from '@tanstack/react-query';
import { invalidate, observeQueryRefresh } from '@/pages/workspace/api/cache-refresh';
import { queries } from '@/shared/api';

test('invalidations coalesce and disposing the page cancels deferred work', async () => {
  queries.clear();
  let calls = 0,
    scheduled = 0,
    pending: (() => void) | undefined;
  const observer = new QueryObserver(queries, {
    queryKey: ['sessions', 'clock'],
    initialData: 0,
    staleTime: Infinity,
    gcTime: Infinity,
    queryFn: async () => ++calls,
  });
  const unsubscribe = observer.subscribe(() => {});
  const stop = observeQueryRefresh((run) => {
    scheduled++;
    pending = run;
    return () => {
      pending = undefined;
    };
  });
  try {
    invalidate(['sessions', 'clock']);
    invalidate(['sessions', 'clock']);
    expect(scheduled).toBe(1);
    expect(calls).toBe(0);
    pending?.();
    await queries.refetchQueries(
      { queryKey: ['sessions', 'clock'], exact: true },
      { cancelRefetch: false }
    );
    expect(calls).toBe(1);
    invalidate(['sessions', 'clock']);
    expect(scheduled).toBe(2);
    stop();
    expect(pending).toBeUndefined();
  } finally {
    stop();
    unsubscribe();
    observer.destroy();
    queries.clear();
  }
});

test('a stale in-flight response cannot erase a later invalidation', async () => {
  queries.clear();
  let calls = 0,
    pending: (() => void) | undefined,
    release: (value: number) => void = () => {};
  const held = new Promise<number>((resolve) => {
    release = resolve;
  });
  const observer = new QueryObserver(queries, {
    queryKey: ['sessions', 'overlap'],
    initialData: 0,
    staleTime: Infinity,
    gcTime: Infinity,
    queryFn: () => (++calls === 1 ? held : Promise.resolve(2)),
  });
  const unsubscribe = observer.subscribe(() => {});
  const stop = observeQueryRefresh((run) => {
    pending = run;
    return () => {
      pending = undefined;
    };
  });
  try {
    const first = observer.refetch();
    invalidate(['sessions', 'overlap']);
    release(1);
    await first;
    pending?.();
    expect(calls).toBe(2);
    await queries.refetchQueries(
      { queryKey: ['sessions', 'overlap'], exact: true },
      { cancelRefetch: false }
    );
  } finally {
    stop();
    unsubscribe();
    observer.destroy();
    queries.clear();
  }
});

test('busy reads receive one follow-up without cancellation or repeated waiters', async () => {
  queries.clear();
  let calls = 0,
    scheduled = 0,
    pending: (() => void) | undefined,
    release: (value: number) => void = () => {};
  const held = new Promise<number>((resolve) => {
    release = resolve;
  });
  const observer = new QueryObserver(queries, {
    queryKey: ['sessions', 'busy'],
    initialData: 0,
    staleTime: Infinity,
    gcTime: Infinity,
    queryFn: () => (++calls === 1 ? held : Promise.resolve(2)),
  });
  const unsubscribe = observer.subscribe(() => {});
  const stop = observeQueryRefresh((run) => {
    scheduled++;
    pending = run;
    return () => {
      pending = undefined;
    };
  });
  const tick = () => {
    const run = pending;
    pending = undefined;
    run?.();
  };
  try {
    const first = observer.refetch();
    invalidate(['sessions', 'busy']);
    tick();
    invalidate(['sessions', 'busy']);
    invalidate(['sessions', 'busy']);
    expect(calls).toBe(1);
    expect(scheduled).toBe(1);
    release(1);
    await first;
    expect(scheduled).toBe(2);
    tick();
    expect(calls).toBe(2);
    await queries.refetchQueries(
      { queryKey: ['sessions', 'busy'], exact: true },
      { cancelRefetch: false }
    );
  } finally {
    stop();
    unsubscribe();
    observer.destroy();
    queries.clear();
  }
});
