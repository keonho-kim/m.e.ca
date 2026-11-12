// Purpose: Select native folders only for loopback browser connections, with an explicit modal fallback.
// Pattern: Cancellable desktop HTTP boundary without automatic request replay.
// Usage: The sidebar owns each request; remote clients always keep the folder browser.
// Related: internal/serverapi/ui_folder.go, webui/src/pages/workspace/model/workspace-picker.ts.
import { nativeRequest, readNativeJSON } from '@/shared/api';
import { failure } from '@/shared/lib/errors';

export async function nativeFolder(hostname: string, signal: AbortSignal) {
  if (!['localhost', '127.0.0.1', '[::1]', '::1'].includes(hostname.toLowerCase()))
    return { kind: 'fallback' } as const;
  const response = await nativeRequest('/v1/ui/pick-folder', { method: 'POST', signal });
  if (response.status === 503 || response.status === 403) {
    await response.body?.cancel();
    return { kind: 'fallback' } as const;
  }
  if (!response.ok) {
    await response.body?.cancel();
    if (response.status === 409) throw failure('folder_picker_busy');
    if (response.status === 500 || response.status === 504)
      throw failure(response.status === 504 ? 'folder_picker_timeout' : 'folder_picker');
    throw failure('network');
  }
  const body = await readNativeJSON(response, signal);
  signal.throwIfAborted();
  if (
    !body ||
    typeof body !== 'object' ||
    !('path' in body) ||
    typeof body.path !== 'string' ||
    body.path.length > 4096
  )
    throw failure('invalid_response');
  return body.path
    ? ({ kind: 'selected', path: body.path } as const)
    : ({ kind: 'canceled' } as const);
}
