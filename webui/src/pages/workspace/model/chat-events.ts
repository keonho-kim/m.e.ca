// Purpose: Classify public response events and reconcile live tool calls with saved results.
// Pattern: Pure chat event projection.
// Usage: History, live activity and stream handoff share the same event identities.
// Related: webui/src/pages/workspace/model/history-rows.ts, webui/src/pages/workspace/ui/chat/live-output.tsx.
import type { EventJson, HistoryItemJson } from '@/shared/api';

export function isResponseEvent(event: EventJson) {
  return ['progress', 'team_progress', 'team_message', 'team_discussion'].includes(
    event.kind ?? ''
  );
}
export function isHistoryBoundary(event: EventJson) {
  return ['progress', 'team_progress', 'tool_started'].includes(event.kind ?? '');
}
export function toolIdentity(event: EventJson) {
  return event.runId && event.tool?.callId
    ? JSON.stringify([event.runId, event.agent ?? '', event.tool.callId])
    : undefined;
}
export function liveActivity(events: EventJson[], history: HistoryItemJson[]) {
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
    return !key || (!saved.has(key) && latest.get(key) === event);
  });
}
