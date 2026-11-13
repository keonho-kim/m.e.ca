// Purpose: Project saved responses, reasoning and one row per tool call into chronological chat surfaces.
// Pattern: Pure chronological history projection.
// Usage: Chat virtualizes these groups rather than switching presentation at tool boundaries.
// Related: webui/src/pages/workspace/ui/chat/chat.tsx, webui/src/pages/workspace/ui/chat/history-row.tsx.
import { toolBatchIdentity, toolIdentity } from '@/pages/workspace/model/chat-events';
import type {
  DetailRequestJson,
  HistoryItemJson,
  RunCompletionJson,
  TranscriptMessageJson,
} from '@/shared/api';
export type HistoryJump = { source: DetailRequestJson; serial: number };
export type HistoryMessageItem = Omit<HistoryItemJson, 'completion' | 'message'> &
  TranscriptMessageJson;
export function historyMessage(item: HistoryItemJson): HistoryMessageItem {
  return {
    sequence: item.sequence,
    truncated: item.truncated,
    event: item.event,
    ...(item.message ?? item.event?.message),
  };
}
export type HistoryRowValue =
  | { item: HistoryMessageItem; thinking?: HistoryMessageItem; tools?: HistoryMessageItem[] }
  | { sequence?: string; completion: RunCompletionJson };
export function historyRows(items: HistoryItemJson[]): HistoryRowValue[] {
  const result: HistoryRowValue[] = [];
  const tools = new Map<string, { row: number; member?: number }>();
  const batches = new Map<string, number>();
  for (const original of items) {
    const { completion } = original;
    const item = historyMessage(original);
    if (completion) {
      result.push({ sequence: original.sequence, completion });
      continue;
    }
    const event = item.event;
    if (event && !event.message && !event.work && !event.notice && !event.tool) continue;
    if (!event && !original.message) continue;
    const key = event && toolIdentity(event);
    if (key) {
      const position = tools.get(key);
      if (position) {
        const row = result[position.row];
        if (row && 'item' in row && row.tools && position.member !== undefined) {
          row.tools[position.member] = item;
          row.item = row.tools[0]!;
        } else result[position.row] = { item };
        continue;
      }
      const batch = toolBatchIdentity(event);
      const index = batch && batches.get(batch);
      if (typeof index === 'number') {
        const row = result[index];
        if (row && 'item' in row && row.tools) {
          tools.set(key, { row: index, member: row.tools.length });
          row.tools.push(item);
          continue;
        }
      }
      if (batch) {
        batches.set(batch, result.length);
        tools.set(key, { row: result.length, member: 0 });
        result.push({ item, tools: [item] });
        continue;
      }
      tools.set(key, { row: result.length });
    }
    const previous = result.at(-1);
    if (
      item.role === 'assistant' &&
      previous &&
      'item' in previous &&
      previous.item.event?.message?.thinking &&
      !previous.thinking &&
      (previous.item.event.agent ?? '') === (item.event?.agent ?? '')
    ) {
      result[result.length - 1] = { item, thinking: previous.item };
    } else result.push({ item });
  }
  return result;
}
