// Purpose: Expose native commands and attachment operations through the shared HTTP adapter.
// Pattern: Same-origin HTTP adapter.
// Usage: Query functions, mutations and explicit image uploads call this module.
// Related: api/meca/v1/control.proto, webui/src/shared/lib/errors/index.ts.

import type { CommandJson, ReplyJson } from '@/shared/api';
import { nativeJSON, nativeRequest, readNativeJSON } from '@/shared/api/http';
import { commandResult, parseReply, replyFailure } from '@/shared/api/reply';
import { failure } from '@/shared/lib/errors';
export async function call(
  workspace: string,
  operation: CommandJson,
  signal?: AbortSignal
): Promise<ReplyJson> {
  const expected = commandResult(operation),
    id = crypto.randomUUID();
  const { response, body } = await nativeJSON('/v1/command', {
    method: 'POST',
    json: {
      ...operation,
      id,
      ...(workspace ? { workspaceId: workspace } : {}),
    },
    signal,
  });
  const reply = parseReply(body, expected, workspace, id);
  if (!response.ok) throw failure('network');
  return reply;
}
export async function control(
  workspace: string,
  task: string,
  operation: 'cancel' | 'approve',
  allow?: boolean,
  approvalId?: string
) {
  const { response, body } = await nativeJSON(
    `/v1/workspaces/${encodeURIComponent(workspace)}/tasks/${encodeURIComponent(task)}/${operation === 'approve' ? 'approvals/' + encodeURIComponent(approvalId ?? '') : operation}`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      ...(operation === 'approve' ? { json: { allow } } : {}),
    }
  );
  const reply = parseReply(body, 'ok', workspace);
  if (!response.ok) throw failure('network');
  return reply;
}
export async function upload(
  workspace: string,
  file: File,
  signal?: AbortSignal
): Promise<{ id: string; expiresAt?: string }> {
  const { response, body } = await nativeJSON('/v1/attachments', {
    method: 'POST',
    headers: { 'Content-Type': file.type, 'X-Meca-Workspace': workspace },
    body: file,
    signal,
  });
  if (!body || typeof body !== 'object') throw failure('invalid_response');
  if (!response.ok) throw replyFailure(body);
  if (!('id' in body) || typeof body.id !== 'string' || !body.id) throw failure('invalid_response');
  return { id: body.id };
}
export async function releaseUpload(workspace: string, id: string) {
  const response = await nativeRequest('/v1/attachments/' + encodeURIComponent(id), {
    method: 'DELETE',
    headers: { 'X-Meca-Workspace': workspace },
    signal: AbortSignal.timeout(5000),
  });
  await response.body?.cancel();
}
export async function loadThumbnail(
  workspace: string,
  session: string,
  sequence: string,
  index: number,
  signal: AbortSignal
) {
  const response = await nativeRequest(
    imageURL(workspace, session, sequence, index) + '?thumbnail=1',
    {
      signal,
    }
  );
  if (!response.ok) {
    throw replyFailure(await readNativeJSON(response, signal));
  }
  const body = await response.blob();
  if (body.type !== 'image/png' || body.size > 1024 * 1024) throw failure('invalid_response');
  return body;
}
export function imageURL(workspace: string, session: string, sequence: string, index: number) {
  return (
    '/v1/workspaces/' +
    encodeURIComponent(workspace) +
    '/sessions/' +
    encodeURIComponent(session) +
    '/images/' +
    sequence +
    '/' +
    index
  );
}
