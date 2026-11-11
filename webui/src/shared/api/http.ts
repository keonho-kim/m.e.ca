// Purpose: Own same-origin fetch, safe transport failures and cancellable JSON decoding.
// Pattern: Native browser HTTP adapter with no automatic replay.
// Usage: Command, attachment and authentication adapters share these transport rules.
// Related: webui/src/shared/api/client.ts, webui/src/features/server-access/api/access.ts.
import { failure } from '@/shared/lib/errors';

type NativeOptions = RequestInit & { json?: unknown };
export async function nativeRequest(path: string, options: NativeOptions = {}) {
  const { json, ...init } = options;
  init.signal?.throwIfAborted();
  const headers = new Headers(init.headers);
  headers.set('X-Meca-UI', '1');
  if (json !== undefined) {
    headers.set('Content-Type', 'application/json');
    init.body = JSON.stringify(json);
  }
  let response: Response;
  try {
    response = await fetch(path, { ...init, headers, credentials: 'same-origin' });
  } catch {
    init.signal?.throwIfAborted();
    throw failure('network');
  }
  init.signal?.throwIfAborted();
  if (response.status === 401) {
    try {
      await response.body?.cancel();
    } catch {
      // The observed denial remains authoritative if the unused body is already disconnected.
    }
    throw failure('unauthorized');
  }
  return response;
}
export async function readNativeJSON(
  response: Response,
  signal?: AbortSignal | null
): Promise<unknown> {
  try {
    return await response.json();
  } catch {
    signal?.throwIfAborted();
    throw failure('invalid_response');
  }
}
export async function nativeJSON(path: string, options: NativeOptions = {}) {
  const response = await nativeRequest(path, options);
  return { response, body: await readNativeJSON(response, options.signal) };
}
