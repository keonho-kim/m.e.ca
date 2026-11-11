// Purpose: Correlate file-change notifications with replies whose reference snapshot is already authoritative.
// Pattern: Bounded reference-only acknowledgement set.
// Usage: File mutations remember keys; event projection consumes them; target deletion and logout clear them.
// Related: webui/src/pages/workspace/model/file-mutations.ts, webui/src/pages/workspace/api/event-cache.ts.
import { type Target, targetKey } from '@/pages/workspace/model/types';

const replies = new Map<string, string>();
export function rememberMutation(target: Target, key: string) {
  replies.set(key, targetKey(target));
  if (replies.size > 128) {
    const oldest = replies.keys().next().value;
    if (oldest !== undefined) replies.delete(oldest);
  }
}
export function forgetMutation(key: string) {
  replies.delete(key);
}
export function consumeMutation(target: Target, key?: string) {
  if (!key || replies.get(key) !== targetKey(target)) return false;
  replies.delete(key);
  return true;
}
export function clearMutationCorrelations(target?: Target) {
  if (!target) {
    replies.clear();
    return;
  }
  const owner = targetKey(target);
  for (const [key, value] of replies) if (value === owner) replies.delete(key);
}
