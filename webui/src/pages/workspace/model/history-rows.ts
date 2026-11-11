// Purpose: Keep persisted reasoning and its adjacent assistant text in the same message surface.
// Pattern: Pure chronological history projection.
// Usage: Chat virtualizes these groups rather than switching presentation at tool boundaries.
// Related: webui/src/pages/workspace/ui/chat/chat.tsx, webui/src/pages/workspace/ui/chat/history-row.tsx.
import type { HistoryItemJson } from '@/shared/api';
export type HistoryRowValue = { item: HistoryItemJson; thinking?: HistoryItemJson };
export function historyRows(items: HistoryItemJson[]): HistoryRowValue[] {
  const result: HistoryRowValue[] = [];
  for (const item of items) {
    const previous = result.at(-1);
    if (
      item.role === 'assistant' &&
      previous?.item.event?.thinking &&
      !previous.thinking &&
      (previous.item.event.agent ?? '') === (item.event?.agent ?? '')
    ) {
      result[result.length - 1] = { item, thinking: previous.item };
    } else result.push({ item });
  }
  return result;
}
