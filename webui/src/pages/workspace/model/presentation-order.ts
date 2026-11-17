// Purpose: Order immutable display identities without coupling them to commit or delivery time.
// Pattern: Pure bounded timeline ordering.
// Usage: Saved history and mixed live history use the same request groups and positions.
// Related: internal/agent/presentation/order.go, webui/src/pages/workspace/model/live-timeline.ts.
export function presentationGroup(value?: string) {
  return value?.split('.')[0] ?? '';
}
export function orderPresentation<T>(values: T[], position: (value: T) => string | undefined): T[] {
  const groups = new Map<string, number>();
  const entries = values.map((value, index) => {
    const id = position(value);
    const group = presentationGroup(id);
    if (group && !groups.has(group)) groups.set(group, index);
    return { value, index, position: id, rank: groups.get(group) ?? index };
  });
  return entries
    .sort((a, b) => {
      if (a.rank !== b.rank) return a.rank - b.rank;
      if (a.position && b.position && a.position !== b.position)
        return a.position < b.position ? -1 : 1;
      return a.index - b.index;
    })
    .map(({ value }) => value);
}
