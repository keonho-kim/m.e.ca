// Purpose: Interleave live generations and activity by their original publication positions.
// Pattern: Pure bounded timeline projection with stable lifecycle identities.
// Usage: LiveOutput renders this sequence; completed tools retain their start position.
// Related: webui/src/pages/workspace/model/chat-events.ts, webui/src/pages/workspace/model/stream-projection.ts.
import {
  liveActivity,
  toolBatchIdentity,
  toolGroups,
  toolIdentity,
} from '@/pages/workspace/model/chat-events';
import type { Draft } from '@/pages/workspace/model/types';
import type { EventJson, HistoryItemJson } from '@/shared/api';

export type LiveEntry =
  | { kind: 'draft'; key: string; draft: Draft; sequence: bigint }
  | { kind: 'activity'; key: string; events: EventJson[]; sequence: bigint };

const sequence = (value?: string) => BigInt(value ?? '0');
export function liveTimeline(
  drafts: Draft[],
  events: EventJson[],
  history: HistoryItemJson[],
  taskOrder: string[] = []
): LiveEntry[] {
  const positions = new Map<string, bigint>();
  for (const event of events) {
    const key = toolBatchIdentity(event) ?? toolIdentity(event);
    if (key && !positions.has(key)) positions.set(key, sequence(event.sequence));
  }
  const entries: LiveEntry[] = drafts
    .filter((draft) => draft.text || draft.thinking)
    .map((draft) => ({
      kind: 'draft',
      key: `draft/${draft.task}/${draft.agent}/${draft.generation}`,
      draft,
      sequence: sequence(draft.startSequence),
    }));
  for (const group of toolGroups(liveActivity(events, history)).slice(-24)) {
    const event = group[0];
    if (!event) continue;
    const identity = toolBatchIdentity(event) ?? toolIdentity(event);
    entries.push({
      kind: 'activity',
      events: group,
      key:
        identity ??
        `event/${event.taskId}/${event.sequence ?? event.historySequence ?? event.cursor}`,
      sequence: (identity && positions.get(identity)) || sequence(event.sequence),
    });
  }
  const tasks = new Map(taskOrder.map((task, index) => [task, index]));
  const task = (entry: LiveEntry) =>
    entry.kind === 'draft' ? entry.draft.task : (entry.events[0]?.taskId ?? '');
  return entries.sort((a, b) => {
    const order = (tasks.get(task(a)) ?? 0) - (tasks.get(task(b)) ?? 0);
    return order || (a.sequence < b.sequence ? -1 : a.sequence > b.sequence ? 1 : 0);
  });
}
