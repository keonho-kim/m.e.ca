// Purpose: Bound inactive server data by retained bytes and query count within the page lifetime.
// Pattern: Page-owned cache eviction and refresh observation.
// Usage: WorkspaceShell mounts both subscriptions and disposes their timers together.
// Related: webui/src/pages/workspace/api/cache-refresh.ts.
import { observeQueryRefresh } from '@/pages/workspace/api/cache-refresh';
import { queries } from '@/shared/api';

const measured = new WeakMap<object, number>();
function retainedBytes(value: unknown, seen = new Set<object>()): number {
  if (typeof value === 'string') return value.length * 2;
  if (!value || typeof value !== 'object' || seen.has(value)) return 16;
  if (value instanceof Blob) return value.size;
  const cached = measured.get(value);
  if (cached !== undefined) return cached;
  seen.add(value);
  let bytes = 32;
  for (const field of Object.values(value)) bytes += retainedBytes(field, seen);
  measured.set(value, bytes);
  return bytes;
}
const kinds = new Set([
  'history',
  'state',
  'session-usage',
  'sessions',
  'directory',
  'skills',
  'skill',
  'preview',
  'original',
  'thumbnail',
]);
export function observeConversationCache() {
  const stopRefresh = observeQueryRefresh();
  let pruning = false;
  const stop = queries.getQueryCache().subscribe((event) => {
    if (pruning) return;
    if (
      event.type !== 'observerRemoved' &&
      !(event.type === 'updated' && event.action.type === 'success')
    )
      return;
    if (event.query.getObserversCount() > 0 || !kinds.has(String(event.query.queryKey[0]))) return;
    const inactive = queries
      .getQueryCache()
      .getAll()
      .filter(
        (query) =>
          query.getObserversCount() === 0 &&
          query.state.fetchStatus === 'idle' &&
          query.state.dataUpdatedAt > 0 &&
          kinds.has(String(query.queryKey[0]))
      );
    let bytes = inactive.reduce((total, query) => total + retainedBytes(query.state.data), 0);
    let count = inactive.length;
    if (count <= 12 && bytes <= 16 * 1024 * 1024) return;
    pruning = true;
    inactive.sort((left, right) => left.state.dataUpdatedAt - right.state.dataUpdatedAt);
    try {
      for (const query of inactive) {
        if (count <= 12 && bytes <= 16 * 1024 * 1024) break;
        bytes -= retainedBytes(query.state.data);
        count--;
        queries.removeQueries({ queryKey: query.queryKey, exact: true });
      }
    } finally {
      pruning = false;
    }
  });
  return () => {
    stop();
    stopRefresh();
  };
}
