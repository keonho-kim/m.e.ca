// Purpose: Interleave saved rows, live generations and activity at stable display positions.
// Pattern: Pure bounded timeline projection with stable lifecycle identities.
// Usage: LiveOutput renders this sequence; completed tools retain their start position.
// Related: webui/src/pages/workspace/model/chat-events.ts, webui/src/pages/workspace/model/stream-projection.ts.
import {
  liveActivity,
  toolGroupIdentity,
  toolGroups,
  toolIdentity,
} from '@/pages/workspace/model/chat-events';
import { type HistoryRowValue, historyRows } from '@/pages/workspace/model/history-rows';
import { orderPresentation } from '@/pages/workspace/model/presentation-order';
import type { Draft } from '@/pages/workspace/model/types';
import type { EventJson, HistoryItemJson } from '@/shared/api';

export type LiveEntry =
  | { kind: 'history'; key: string; row: HistoryRowValue; sequence: bigint }
  | { kind: 'draft'; key: string; draft: Draft; sequence: bigint }
  | { kind: 'activity'; key: string; events: EventJson[]; sequence: bigint };

const sequence = (value?: string) => BigInt(value ?? '0');
export function liveTimeline(
  drafts: Draft[],
  events: EventJson[],
  history: HistoryItemJson[],
  taskOrder: string[] = [],
  savedRows: HistoryRowValue[] = historyRows(history)
): LiveEntry[] {
  const positions = new Map<string, bigint>();
  for (const event of events) {
    const key = toolGroupIdentity(event) ?? toolIdentity(event);
    if (key && !positions.has(key)) positions.set(key, sequence(event.sequence));
  }
  const entries: LiveEntry[] = drafts
    .filter((draft) => draft.text || draft.thinking)
    .map((draft) => ({
      kind: 'draft',
      key: draft.presentation ?? `draft/${draft.task}/${draft.agent}/${draft.generation}`,
      draft,
      sequence: sequence(draft.startSequence),
    }));
  for (const group of toolGroups(liveActivity(events, history)).slice(-24)) {
    const event = group[0];
    if (!event) continue;
    const identity = toolGroupIdentity(event) ?? toolIdentity(event);
    entries.push({
      kind: 'activity',
      events: group,
      key:
        identity ??
        `event/${event.taskId}/${event.sequence ?? event.historySequence ?? event.cursor}`,
      sequence: (identity && positions.get(identity)) || sequence(event.sequence),
    });
  }
  const occupied = new Set(entries.map((entry) => entry.key));
  const saved = savedRows
    .map((row): LiveEntry => {
      const item = 'item' in row ? row.item : undefined;
      return {
        kind: 'history',
        row,
        key:
          (item?.event && (toolGroupIdentity(item.event) ?? toolIdentity(item.event))) ??
          item?.presentation ??
          `history/${item?.sequence ?? ('sequence' in row ? row.sequence : '')}`,
        sequence: sequence(item?.sequence ?? ('sequence' in row ? row.sequence : undefined)),
      };
    })
    .filter((entry) => !occupied.has(entry.key));
  const tasks = new Map(taskOrder.map((task, index) => [task, index]));
  const task = (entry: LiveEntry) =>
    entry.kind === 'draft'
      ? entry.draft.task
      : entry.kind === 'activity'
        ? (entry.events[0]?.taskId ?? '')
        : '';
  entries.sort((a, b) => {
    const order = (tasks.get(task(a)) ?? 0) - (tasks.get(task(b)) ?? 0);
    return order || (a.sequence < b.sequence ? -1 : a.sequence > b.sequence ? 1 : 0);
  });
  return orderPresentation([...saved, ...entries], (entry) =>
    entry.kind === 'draft'
      ? entry.draft.presentation
      : entry.kind === 'activity'
        ? entry.events[0]?.presentation
        : 'item' in entry.row
          ? (entry.row.item.presentation ?? entry.row.item.event?.presentation)
          : undefined
  );
}
