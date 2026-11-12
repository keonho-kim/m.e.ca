// Purpose: Compare committed transcript positions and discard duplicate or stale replay frames.
// Pattern: Pure ordering decisions independent of transport and React.
// Usage: Stream projection and history acknowledgement share these checks.
// Related: webui/src/pages/workspace/model/stream-projection.ts, webui/src/pages/workspace/model/stream-slice.ts.
import type { EventJson } from '@/shared/api';

export function reached(through: string | undefined, required: string | undefined) {
  return !required || required === '0' || BigInt(through ?? '0') >= BigInt(required);
}
export function orderedEvents(batch: EventJson[], previous: string) {
  let cursor = previous,
    filtered: EventJson[] | undefined;
  for (const [index, event] of batch.entries()) {
    const next = event.cursor;
    let accepted = true;
    if (next && cursor) {
      const [epoch, sequence] = next.split(':'),
        [oldEpoch, oldSequence] = cursor.split(':');
      accepted = epoch === oldEpoch && BigInt(sequence ?? '0') > BigInt(oldSequence ?? '0');
    }
    if (!accepted) filtered ??= batch.slice(0, index);
    else {
      if (next) cursor = next;
      filtered?.push(event);
    }
  }
  return { events: filtered ?? batch, cursor };
}
