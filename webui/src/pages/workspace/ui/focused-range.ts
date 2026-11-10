// Purpose: Keep focused or edited rows mounted outside a virtual list's visible window.
// Pattern: Pure interaction-preserving range projection.
// Usage: Conversation, skill and queue lists share this policy with TanStack Virtual.
// Related: webui/src/pages/workspace/ui/navigation/session-list.tsx, webui/src/pages/workspace/ui/composer/queue.tsx.
import { defaultRangeExtractor, type Range } from '@tanstack/react-virtual';

export function focusedRange(range: Range, ...retained: number[]) {
  const values = defaultRangeExtractor(range);
  for (const index of retained)
    if (index >= 0 && index < range.count && !values.includes(index)) values.push(index);
  return values.sort((left, right) => left - right);
}
