// Purpose: Send native typed commands and authenticate without retaining server credentials.
// Pattern: Same-origin HTTP adapter.
// Usage: Query functions, mutations and explicit image uploads call this module.
// Related: api/meca/v1/control.proto, webui/src/shared/lib/errors/index.ts.

import type { CommandJson, ReplyJson } from '@/shared/api';
import { ApiError, failure } from '@/shared/lib/errors';
export async function call(
  workspace: string,
  operation: CommandJson,
  signal?: AbortSignal
): Promise<ReplyJson> {
  let response: Response;
  try {
    response = await fetch('/v1/command', {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json', 'X-Meca-UI': '1' },
      body: JSON.stringify({
        ...operation,
        id: crypto.randomUUID(),
        ...(workspace ? { workspaceId: workspace } : {}),
      }),
      signal,
    });
  } catch (error) {
    if (signal?.aborted) throw error;
    throw failure('network');
  }
  if (response.status === 401) throw failure('unauthorized');
  let reply: ReplyJson;
  try {
    reply = (await response.json()) as ReplyJson;
  } catch {
    throw failure('invalid_response');
  }
  if (reply.error)
    throw new ApiError(reply.error.code ?? 'internal', reply.error.message ?? '', true);
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
  const response = await fetch(
    `/v1/workspaces/${encodeURIComponent(workspace)}/tasks/${encodeURIComponent(task)}/${operation === 'approve' ? 'approvals/' + encodeURIComponent(approvalId ?? '') : operation}`,
    {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json', 'X-Meca-UI': '1' },
      ...(operation === 'approve' ? { body: JSON.stringify({ allow }) } : {}),
    }
  );
  if (response.status === 401) throw failure('unauthorized');
  const reply = (await response.json()) as ReplyJson;
  if (reply.error)
    throw new ApiError(reply.error.code ?? 'internal', reply.error.message ?? '', true);
  if (!response.ok) throw failure('network');
  return reply;
}
export async function upload(
  workspace: string,
  file: File,
  signal?: AbortSignal
): Promise<{ id: string; expiresAt?: string }> {
  const response = await fetch('/v1/attachments', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'Content-Type': file.type, 'X-Meca-UI': '1', 'X-Meca-Workspace': workspace },
    body: file,
    signal,
  });
  if (response.status === 401) throw failure('unauthorized');
  const body = await response.json();
  if (!response.ok)
    throw new ApiError(body.error?.code ?? 'upload', body.error?.message ?? body.message ?? '');
  return body;
}
export async function releaseUpload(workspace: string, id: string) {
  await fetch('/v1/attachments/' + encodeURIComponent(id), {
    method: 'DELETE',
    headers: { 'X-Meca-UI': '1', 'X-Meca-Workspace': workspace },
    credentials: 'same-origin',
  });
}
export async function loadThumbnail(
  workspace: string,
  session: string,
  sequence: string,
  index: number,
  signal: AbortSignal
) {
  const response = await fetch(imageURL(workspace, session, sequence, index) + '?thumbnail=1', {
    credentials: 'same-origin',
    signal,
  });
  if (response.status === 401) throw failure('unauthorized');
  if (!response.ok) {
    const reply = (await response.json()) as ReplyJson;
    throw new ApiError(reply.error?.code ?? 'network', reply.error?.message ?? '');
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
