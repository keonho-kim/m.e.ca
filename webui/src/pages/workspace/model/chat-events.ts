// Purpose: Classify public response events and reconcile live tool calls with saved results.
// Pattern: Pure chat event projection.
// Usage: History, live activity and stream handoff share the same event identities.
// Related: webui/src/pages/workspace/model/history-rows.ts, webui/src/pages/workspace/ui/chat/live-output.tsx.

import { terminalTool } from '@/pages/workspace/model/tool-status';
import type { EventJson, HistoryItemJson } from '@/shared/api';

export function isResponseEvent(event: EventJson) {
  return ['progress', 'team_progress', 'team_message', 'team_discussion'].includes(
    event.kind ?? ''
  );
}
export function isHistoryBoundary(event: EventJson) {
  return ['progress', 'team_progress', 'tool_started'].includes(event.kind ?? '');
}
export function toolIdentity(event?: EventJson) {
  return event?.runId && event.tool?.executionId
    ? JSON.stringify([event.runId, event.agent ?? '', event.tool.executionId])
    : undefined;
}
export function toolBatchIdentity(event?: EventJson) {
  return event?.runId && event.tool?.batchId
    ? JSON.stringify([event.runId, event.agent ?? '', event.tool.batchId])
    : undefined;
}
export function toolGroups(events: EventJson[]): EventJson[][] {
  const groups: EventJson[][] = [],
    batches = new Map<string, EventJson[]>();
  for (const event of events) {
    const key = toolBatchIdentity(event),
      existing = key && batches.get(key);
    if (existing) existing.push(event);
    else {
      const group = [event];
      groups.push(group);
      if (key) batches.set(key, group);
    }
  }
  return groups;
}
export function mergeToolEvents(saved: EventJson[], live: EventJson[]) {
  const merged = new Map(saved.map((event) => [toolIdentity(event), event]));
  for (const event of live) {
    const key = toolIdentity(event),
      previous = merged.get(key);
    if (!key) continue;
    if (previous) {
      const before = previous.tool?.reference,
        after = event.tool?.reference;
      const oldSequence = toolSequence(before?.sequence),
        nextSequence = toolSequence(after?.sequence);
      const comparable =
        before?.sessionId &&
        before.sessionId === after?.sessionId &&
        oldSequence !== undefined &&
        nextSequence !== undefined;
      if (comparable && nextSequence <= oldSequence) continue;
      if (terminalTool(previous) && (!terminalTool(event) || !comparable)) continue;
      if (previous.kind !== 'tool_started' && event.kind === 'tool_started') continue;
    }
    merged.set(key, event);
  }
  return [...merged.values()];
}
function toolSequence(value: string | undefined) {
  return value && /^[1-9]\d{0,18}$/.test(value) ? BigInt(value) : undefined;
}
export function liveActivity(events: EventJson[], history: HistoryItemJson[]) {
  const batches = new Set(
    history.flatMap((item) => {
      const key = item.event && toolBatchIdentity(item.event);
      return key ? [key] : [];
    })
  );
  const saved = new Set(
    history.flatMap((item) => {
      const key = item.event && toolIdentity(item.event);
      return key ? [key] : [];
    })
  );
  const latest = new Map<string, EventJson>();
  for (const event of events) {
    const key = toolIdentity(event);
    if (key) latest.set(key, event);
  }
  return events.filter((event) => {
    if (isResponseEvent(event) || event.kind === 'tool_proposed' || event.kind === 'approved')
      return false;
    const key = toolIdentity(event);
    const batch = toolBatchIdentity(event);
    return (
      (!batch || !batches.has(batch)) && (!key || (!saved.has(key) && latest.get(key) === event))
    );
  });
}
