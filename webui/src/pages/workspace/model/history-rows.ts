// Purpose: Project saved responses, reasoning and one row per tool call into chronological chat surfaces.
// Pattern: Pure chronological history projection.
// Usage: Chat virtualizes these groups rather than switching presentation at tool boundaries.
// Related: webui/src/pages/workspace/ui/chat/chat.tsx, webui/src/pages/workspace/ui/chat/history-row.tsx.
import { isResponseEvent, toolIdentity } from '@/pages/workspace/model/chat-events';
import type { HistoryItemJson } from '@/shared/api';
export type HistoryRowValue = { item: HistoryItemJson; thinking?: HistoryItemJson };
export function historyRows(items: HistoryItemJson[]): HistoryRowValue[] {
  const result: HistoryRowValue[] = [];
  const tools = new Map<string, number>();
  for (const original of items) {
    const event = original.event;
    if (event?.kind === 'tool_proposed' || event?.kind === 'approved') continue;
    const item =
      event && isResponseEvent(event)
        ? { ...original, role: event.agent === 'user' ? 'user' : 'assistant', text: event.text }
        : original;
    const key = event && toolIdentity(event);
    if (key) {
      const index = tools.get(key);
      if (index !== undefined) {
        result[index] = { item };
        continue;
      }
      tools.set(key, result.length);
    }
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
