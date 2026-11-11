// Purpose: Admit editor text against the server's UTF-8 byte limit without encoded copies.
// Pattern: Pure boundary calculation.
// Usage: The editor checks native changes and clipboard replacements before recording history.
// Related: webui/src/shared/lib/utf8/index.ts, webui/src/shared/ui/code-editor/index.tsx.
import { utf8Bytes } from '@/shared/lib/utf8';

export function fitsEditorText(text: string, maxBytes: number) {
  if (maxBytes <= 0 || text.length > maxBytes) return false;
  return text.length * 3 <= maxBytes || utf8Bytes(text) <= maxBytes;
}
