// Purpose: Reject oversized logical or JSON-escaped requests before staging attachments.
// Pattern: Cached boundary preflight using server-owned capabilities.
// Usage: Composer and submission apply these limits to immutable captured input.
// Related: webui/src/pages/workspace/model/input-usage.ts, api/meca/v1/workspace.proto.
import { inputBytes } from '@/pages/workspace/model/input-usage';
import type { Input } from '@/pages/workspace/model/types';
import type { CapabilitiesJson } from '@/shared/api';
import { failure } from '@/shared/lib/errors';

const wireSizes = new WeakMap<Input, number>();
function encodedTextBytes(input: Input) {
  const known = wireSizes.get(input);
  if (known !== undefined) return known;
  let extra = 0;
  for (let i = 0; i < input.text.length; i++) {
    const char = input.text.charCodeAt(i);
    if (char === 34 || char === 92) extra++;
    else if (char < 32) extra += [8, 9, 10, 12, 13].includes(char) ? 1 : 5;
    else if (char >= 0xd800 && char <= 0xdfff) {
      if (
        char <= 0xdbff &&
        input.text.charCodeAt(i + 1) >= 0xdc00 &&
        input.text.charCodeAt(i + 1) <= 0xdfff
      )
        i++;
      else extra += 3;
    }
  }
  const bytes = inputBytes(input) + extra;
  wireSizes.set(input, bytes);
  return bytes;
}
export function checkSubmissionLimits(input: Input, capabilities: CapabilitiesJson) {
  const prompt = Number(capabilities.promptBytes ?? 0),
    command = Number(capabilities.commandBytes ?? 0);
  if (prompt <= 0 || command <= 0) throw failure('invalid_response');
  if (inputBytes(input) > prompt) throw failure('input_limit');
  // Only the small envelope is serialized here; the large text is measured without copying it.
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
  if (encodedTextBytes(input) + new TextEncoder().encode(envelope).length > command)
    throw failure('input_limit');
}
