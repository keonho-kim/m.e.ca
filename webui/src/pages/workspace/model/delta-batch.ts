// Purpose: Join adjacent compatible text deltas without crossing ordered control events.
// Pattern: Pure batch compaction.
// Usage: The stream reducer projects each contiguous source update once.
// Related: webui/src/pages/workspace/model/stream-projection.ts.
import type { EventJson } from '@/shared/api';

function compatible(left: EventJson, right: EventJson) {
  return (
    left.kind === 'assistant_delta' &&
    right.kind === left.kind &&
    left.workspaceId === right.workspaceId &&
    left.sessionId === right.sessionId &&
    left.taskId === right.taskId &&
    left.agent === right.agent &&
    left.generation === right.generation &&
    !right.reset
  );
}
export function compactDeltas(events: EventJson[]) {
  const values: EventJson[] = [];
  for (let i = 0; i < events.length; i++) {
    const first = events[i];
    if (!first) continue;
    let last = first,
      text: string[] | undefined,
      thinking: string[] | undefined;
    while (i + 1 < events.length) {
      const next = events[i + 1];
      if (!next || !compatible(last, next)) break;
      text ??= [first.text ?? ''];
      thinking ??= [first.thinking ?? ''];
      text.push(next.text ?? '');
      thinking.push(next.thinking ?? '');
      last = next;
      i++;
    }
    values.push(
      text
        ? { ...last, reset: first.reset, text: text.join(''), thinking: thinking?.join('') }
        : first
    );
  }
  return values;
}
