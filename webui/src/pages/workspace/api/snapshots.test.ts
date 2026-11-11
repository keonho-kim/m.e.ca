// Purpose: Verify shared state reads keep stream bodies outside metadata caches and preserve cancellation ownership.
// Pattern: Controlled request lifecycle regression tests.
// Usage: bun test src/pages/workspace/api/snapshots.test.ts.
// Related: webui/src/pages/workspace/api/snapshots.ts, webui/src/pages/workspace/api/queries.ts.
import { expect, test } from 'bun:test';
import { stateOptions } from '@/pages/workspace/api/queries';
import { readSnapshot } from '@/pages/workspace/api/snapshots';
import { queries } from '@/shared/api';

test('one canceled subscriber leaves the shared metadata and stream read alive', async () => {
  const original = globalThis.fetch;
  let finish!: (response: Response) => void;
  const reply = new Promise<Response>((resolve) => {
    finish = resolve;
  });
  let requests = 0;
  let requestSignal: AbortSignal | undefined;
  const first = new AbortController(),
    second = new AbortController();
  try {
    queries.clear();
    globalThis.fetch = Object.assign(
      async (_url: string | URL | Request, options?: RequestInit) => {
        requests++;
        requestSignal = options?.signal ?? undefined;
        return reply;
      },
      { preconnect: original.preconnect }
    );
    const target = { workspace: 'snapshot', session: 'one' };
    const canceled = readSnapshot(target, first.signal);
    const stream = readSnapshot(target, second.signal);
    const metadata = queries.fetchQuery(stateOptions(target));
    first.abort();
    expect((await Promise.allSettled([canceled]))[0].status).toBe('rejected');
    expect(requestSignal?.aborted).toBe(false);
    finish(
      Response.json({
        state: {
          cursor: 'one',
          drafts: [{ text: 'live body' }],
          activity: [{ text: 'live event' }],
        },
      })
    );
    expect((await stream).drafts?.[0]?.text).toBe('live body');
    expect(await metadata).toEqual({ cursor: 'one' });
    expect(queries.getQueryData(stateOptions(target).queryKey)).toEqual({ cursor: 'one' });
    expect(requests).toBe(1);
  } finally {
    first.abort();
    second.abort();
    queries.clear();
    globalThis.fetch = original;
  }
});
