// Purpose: Bound undo text payloads without losing the current editor record.
// Pattern: Explicit state transition over the editor library's public session contract.
// Usage: The controlled adapter trims history after the library completes each input event.
// Related: webui/src/shared/ui/code-editor/index.tsx, webui/src/shared/ui/code-editor/history.test.ts.
import type { ComponentRef } from 'react';
import type SimpleEditor from 'react-simple-code-editor';

export type SimpleEditorHandle = ComponentRef<typeof SimpleEditor>;
type History = SimpleEditorHandle['session']['history'];
// The library separately bounds metadata to 100 records; two complete 2MiB files still fit.
const bytes = (record: History['stack'][number]) => record.value.length * 2;

export function trimEditorHistory(history: History, budget = 8 * 1024 * 1024) {
  const stack = history.stack;
  let retained = stack.reduce((sum, record) => sum + bytes(record), 0),
    removed = 0;
  while (retained > budget && removed < history.offset) {
    const record = stack[removed];
    if (!record) break;
    retained -= bytes(record);
    removed++;
  }
  if (removed) {
    stack.splice(0, removed);
    history.offset -= removed;
  }
  while (retained > budget && stack.length > history.offset + 1) {
    const record = stack.pop();
    if (!record) break;
    retained -= bytes(record);
  }
}
