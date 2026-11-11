// Purpose: Measure encoded text and preserve Unicode boundaries in bounded display windows.
// Pattern: Pure text-boundary calculations without encoded copies.
// Usage: Composer budgets input bytes; stream projection retains a valid text suffix.
// Related: webui/src/pages/workspace/model/input-usage.ts, webui/src/pages/workspace/model/stream-projection.ts.
export function utf8Bytes(text: string) {
  let size = 0;
  for (let i = 0; i < text.length; i++) {
    const n = text.charCodeAt(i);
    size += n < 128 ? 1 : n < 2048 ? 2 : 3;
    if (n >= 0xd800 && n <= 0xdbff && i + 1 < text.length) {
      const next = text.charCodeAt(i + 1);
      if (next >= 0xdc00 && next <= 0xdfff) {
        size++;
        i++;
      }
    }
  }
  return size;
}

export function textTail(text: string, limit: number) {
  let start = Math.max(0, text.length - limit);
  if (start > 0 && start < text.length) {
    const next = text.charCodeAt(start),
      previous = text.charCodeAt(start - 1);
    if (next >= 0xdc00 && next <= 0xdfff && previous >= 0xd800 && previous <= 0xdbff) start++;
  }
  return text.slice(start);
}
