// Purpose: Measure immutable draft bodies once and charge shared attachment bytes once.
// Pattern: Weak reference accounting for composer-owned values.
// Usage: Composer mutations reconcile budgets before committing input or queue changes.
// Related: webui/src/pages/workspace/model/composer-slice.ts, webui/src/shared/lib/utf8/index.ts.
import type { Input, WorkspaceState } from '@/pages/workspace/model/types';
import { utf8Bytes } from '@/shared/lib/utf8';

const sizes = new WeakMap<Input, number>();
export function inputBytes(input: Input) {
  let bytes = sizes.get(input);
  if (bytes === undefined) {
    bytes = utf8Bytes(input.text);
    sizes.set(input, bytes);
  }
  return bytes;
}
export function rememberInputBytes(input: Input, bytes: number) {
  sizes.set(input, bytes);
}
export function attachmentBytes(state: Pick<WorkspaceState, 'inputs' | 'queues'>) {
  const files = new Set<File>();
  for (const input of Object.values(state.inputs)) for (const file of input.files) files.add(file);
  for (const queue of Object.values(state.queues))
    for (const input of queue) for (const file of input.files) files.add(file);
  let bytes = 0;
  for (const file of files) bytes += file.size;
  return bytes;
}
