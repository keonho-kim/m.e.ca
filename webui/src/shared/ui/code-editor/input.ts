// Purpose: Admit editor text against the server's UTF-8 byte limit without encoded copies.
// Pattern: Pure boundary calculation.
// Usage: The editor checks native changes and clipboard replacements before recording history.
// Related: webui/src/shared/lib/utf8/index.ts, webui/src/shared/ui/code-editor/index.tsx.
export function editorTextAdmission(maxBytes: number) {
  const encoder = new TextEncoder();
  let scratch: Uint8Array | undefined;
  return (text: string) => {
    if (maxBytes <= 0 || text.length > maxBytes) return false;
    if (text.length * 3 <= maxBytes) return true;
    scratch ??= new Uint8Array(64 * 1024);
    let offset = 0,
      bytes = 0;
    while (offset < text.length) {
      const result = encoder.encodeInto(text.slice(offset), scratch);
      offset += result.read;
      bytes += result.written;
      if (bytes > maxBytes || result.read === 0) return false;
    }
    return true;
  };
}
