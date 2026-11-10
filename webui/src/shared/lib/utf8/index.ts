// Purpose: Count UTF-8 input bytes without allocating encoded copies.
// Pattern: Pure byte-budget calculation.
// Usage: Composer and submission measure accepted request text.
// Related: webui/src/pages/workspace/model/input-usage.ts.
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
