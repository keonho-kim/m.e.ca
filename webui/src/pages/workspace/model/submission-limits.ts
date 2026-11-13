// Purpose: Reject oversized logical or escaped requests before staging attachments without blocking input.
// Pattern: Weak immutable preflight cache with cooperative, cancellable large-text scanning.
// Usage: Composer and submission use asynchronous admission; boundary tests also exercise the pure path.
// Related: webui/src/pages/workspace/model/input-usage.ts, webui/src/shared/lib/scheduling/index.ts.
import { inputBytes } from '@/pages/workspace/model/input-usage';
import type { Input } from '@/pages/workspace/model/types';
import type { CapabilitiesJson } from '@/shared/api';
import { failure } from '@/shared/lib/errors';
import { yieldToInput } from '@/shared/lib/scheduling';

const wireSizes = new WeakMap<Input, number>();
function* escapedChunks(text: string) {
  let extra = 0;
  for (let i = 0; i < text.length; i++) {
    const char = text.charCodeAt(i);
    if (char === 34 || char === 92) extra++;
    else if (char < 32)
      extra += char === 8 || char === 9 || char === 10 || char === 12 || char === 13 ? 1 : 5;
    else if (char >= 0xd800 && char <= 0xdfff) {
      if (char <= 0xdbff && text.charCodeAt(i + 1) >= 0xdc00 && text.charCodeAt(i + 1) <= 0xdfff)
        i++;
      else extra += 3;
    }
    if ((i & 65535) === 65535) yield extra;
  }
  yield extra;
}
function textBudget(input: Input, capabilities: CapabilitiesJson) {
  const prompt = Number(capabilities.promptBytes ?? 0),
    command = Number(capabilities.commandBytes ?? 0);
  if (
    !Number.isSafeInteger(prompt) ||
    !Number.isSafeInteger(command) ||
    prompt <= 0 ||
    command <= 0
  )
    throw failure('invalid_response');
  if (inputBytes(input) > prompt) throw failure('input_limit');
  const envelope = JSON.stringify({
    start: {
      sessionId: 'session_'.padEnd(34, 'x'),
      key: ''.padEnd(36, 'x'),
      text: '',
      paths: input.paths,
      attachmentIds: input.files.map(() => 'img_'.padEnd(30, 'x')),
    },
    id: ''.padEnd(36, 'x'),
    workspaceId: 'workspace_'.padEnd(74, 'x'),
  });
  return command - new TextEncoder().encode(envelope).length;
}
export function checkSubmissionLimits(input: Input, capabilities: CapabilitiesJson) {
  const limit = textBudget(input, capabilities),
    base = inputBytes(input),
    known = wireSizes.get(input);
  if (known !== undefined) {
    if (known > limit) throw failure('input_limit');
    return;
  }
  let bytes = base;
  for (const extra of escapedChunks(input.text)) {
    bytes = base + extra;
    if (bytes > limit) throw failure('input_limit');
  }
  wireSizes.set(input, bytes);
}
export async function admitSubmission(
  input: Input,
  capabilities: CapabilitiesJson,
  signal?: AbortSignal
) {
  signal?.throwIfAborted();
  if (input.text.length < 65536 || wireSizes.has(input)) {
    checkSubmissionLimits(input, capabilities);
    return;
  }
  const limit = textBudget(input, capabilities),
    base = inputBytes(input);
  let bytes = base,
    deadline = performance.now() + 4;
  for (const extra of escapedChunks(input.text)) {
    bytes = base + extra;
    if (bytes > limit) throw failure('input_limit');
    if (performance.now() >= deadline) {
      await yieldToInput(signal);
      deadline = performance.now() + 4;
    }
  }
  signal?.throwIfAborted();
  wireSizes.set(input, bytes);
}
