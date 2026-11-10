// Purpose: Coalesce scoped invalidations without losing facts that arrive during a read.
// Pattern: Page-owned refresh queue with one deferred follow-up per in-flight query.
// Usage: Mutation and event adapters call invalidate; the workspace owns cancellation.
// Related: webui/src/pages/workspace/api/queries.ts, webui/src/pages/workspace/api/cache-policy.ts.
import { queries } from '@/shared/api';

type Delay = (run: () => void) => () => void;
type Refresh = (hashes: string[]) => void;
const owners = new Set<Refresh>();
const delay: Delay = (run) => {
  const timer = setTimeout(run, 50);
  return () => clearTimeout(timer);
};
export function observeQueryRefresh(schedule: Delay = delay) {
  const pending = new Set<string>(),
    waiting = new Set<string>();
  let cancel: (() => void) | undefined,
    closed = false;
  const enqueue: Refresh = (hashes) => {
    if (closed) return;
    for (const hash of hashes) if (!waiting.has(hash)) pending.add(hash);
    if (cancel || !pending.size) return;
    cancel = schedule(() => {
      cancel = undefined;
      const hashes = new Set(pending),
        ready = new Set<string>();
      pending.clear();
      for (const query of queries
        .getQueryCache()
        .findAll({ type: 'active', predicate: (query) => hashes.has(query.queryHash) })) {
        const hash = query.queryHash;
        if (query.state.fetchStatus !== 'idle' && query.promise) {
          if (waiting.has(hash)) continue;
          waiting.add(hash);
          const resume = () => {
            waiting.delete(hash);
            enqueue([hash]);
          };
          void query.promise.then(resume, resume);
        } else ready.add(hash);
      }
      if (ready.size)
        void queries.refetchQueries(
          { type: 'active', predicate: (query) => ready.has(query.queryHash) },
          { cancelRefetch: false }
        );
    });
  };
  owners.add(enqueue);
  return () => {
    closed = true;
    owners.delete(enqueue);
    cancel?.();
    pending.clear();
    waiting.clear();
  };
}

export function invalidate(queryKey: readonly unknown[]) {
  if (!owners.size) {
    void queries.invalidateQueries({ queryKey });
    return;
  }
  const active: string[] = [];
  for (const query of queries.getQueryCache().findAll({ queryKey })) {
    query.invalidate();
    if (query.isActive()) active.push(query.queryHash);
  }
  for (const refresh of owners) refresh(active);
}
