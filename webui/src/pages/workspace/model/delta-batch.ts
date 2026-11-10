// Purpose: Coalesce interleaved text deltas once per source while retaining control and generation order.
// Pattern: Pure batch compaction with ordered barriers.
// Usage: The stream reducer projects text after transport delivery.
// Related: webui/src/pages/workspace/model/stream-projection.ts.
import type { EventJson } from '@/shared/api';

type Run = { index: number; last: EventJson; text?: string[]; thinking?: string[] };

function sourceKey(event: EventJson) {
  return `${event.workspaceId}/${event.sessionId}/${event.taskId}/${event.agent ?? ''}`;
}

export function compactDeltas(events: EventJson[]) {
  if (events.length < 2) return events;
  const values: EventJson[] = [];
  const active = new Map<string, Run>();
  const runs: Run[] = [];
  const finish = () => {
    for (const run of runs) {
      if (run.text) {
        values[run.index] = {
          ...run.last,
          text: run.text.join(''),
          thinking: run.thinking?.join(''),
        };
      }
    }
    runs.length = 0;
    active.clear();
  };
  for (const event of events) {
    // Statistics update the selected conversation in arrival order. A reset
    // or control event observes exactly the text preceding it.
    if (event.kind !== 'assistant_delta' || event.reset || event.statistics) {
      finish();
      values.push(event);
      continue;
    }
    const key = sourceKey(event);
    const previous = active.get(key);
    if (previous && previous.last.generation === event.generation) {
      previous.text ??= [previous.last.text ?? ''];
      previous.thinking ??= [previous.last.thinking ?? ''];
      previous.text.push(event.text ?? '');
      previous.thinking.push(event.thinking ?? '');
      previous.last = event;
      continue;
    }
    const run = { index: values.length, last: event };
    runs.push(run);
    active.set(key, run);
    values.push(event);
  }
  finish();
  return values;
}
