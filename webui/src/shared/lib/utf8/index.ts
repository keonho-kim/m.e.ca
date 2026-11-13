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

export type TextEdit = { start: number; end: number; inserted: string };

// Native beforeinput ranges avoid searching the unchanged prefix on each key.
// Padding protects surrogate pairs formed or broken at the replacement edges.
export function editedBytes(
  previous: string,
  next: string,
  previousBytes: number,
  edit?: TextEdit
) {
  if (
    edit &&
    Number.isInteger(edit.start) &&
    Number.isInteger(edit.end) &&
    edit.start >= 0 &&
    edit.end >= edit.start &&
    edit.end <= previous.length &&
    next.length === previous.length - (edit.end - edit.start) + edit.inserted.length &&
    next.slice(edit.start, edit.start + edit.inserted.length) === edit.inserted
  ) {
    const start = Math.max(0, edit.start - 1);
    const padding = Math.min(1, previous.length - edit.end);
    return (
      previousBytes -
      utf8Bytes(previous.slice(start, edit.end + padding)) +
      utf8Bytes(next.slice(start, edit.start + edit.inserted.length + padding))
    );
  }
  if (previous === next) return previousBytes;
  // IME, bulk replacement and complex token DOM require a complete check.
  // Searching large strings for a change can cost more than this single scan.
  if (Math.max(previous.length, next.length) > 65536) return utf8Bytes(next);
  const common = Math.min(previous.length, next.length);
  let prefix = 0;
  if (
    (next.length > previous.length && next.startsWith(previous)) ||
    (previous.length > next.length && previous.startsWith(next))
  )
    prefix = common;
  else {
    while (
      prefix + 4096 <= common &&
      next.startsWith(previous.slice(prefix, prefix + 4096), prefix)
    )
      prefix += 4096;
    while (prefix < common && previous.charCodeAt(prefix) === next.charCodeAt(prefix)) prefix++;
  }
  let suffix = 0;
  while (
    suffix + 4096 <= common - prefix &&
    next.startsWith(
      previous.slice(previous.length - suffix - 4096, previous.length - suffix),
      next.length - suffix - 4096
    )
  )
    suffix += 4096;
  while (
    suffix < common - prefix &&
    previous.charCodeAt(previous.length - suffix - 1) === next.charCodeAt(next.length - suffix - 1)
  )
    suffix++;
  const start = Math.max(0, prefix - 1),
    padding = Math.min(1, suffix);
  return (
    previousBytes -
    utf8Bytes(previous.slice(start, previous.length - suffix + padding)) +
    utf8Bytes(next.slice(start, next.length - suffix + padding))
  );
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
