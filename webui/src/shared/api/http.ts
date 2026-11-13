// Purpose: Own same-origin fetch, safe transport failures and cancellable JSON decoding.
// Pattern: Native browser HTTP adapter with no automatic replay.
// Usage: Command, attachment and authentication adapters share these transport rules.
// Related: webui/src/shared/api/client.ts, webui/src/features/server-access/api/access.ts.

import { requestAccessAllowed, requestAccessSignal } from '@/shared/api/access-lifetime';
import { jsonBody } from '@/shared/api/json-body';
import { failure } from '@/shared/lib/errors';

type NativeOptions = RequestInit & { json?: unknown; readOnly?: boolean };
export async function nativeRequest(path: string, options: NativeOptions = {}) {
  const { json, readOnly: _readOnly, ...init } = options;
  if (!['/v1/ui/session', '/v1/ui/login'].includes(path)) {
    if (!requestAccessAllowed()) throw failure('unauthorized');
    init.signal = AbortSignal.any([requestAccessSignal(), ...(init.signal ? [init.signal] : [])]);
  }
  init.signal?.throwIfAborted();
  const headers = new Headers(init.headers);
  headers.set('X-Meca-UI', '1');
  if (json !== undefined) {
    headers.set('Content-Type', 'application/json');
    init.body = await jsonBody(json, init.signal);
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
    const body: unknown = await response.json();
    signal?.throwIfAborted();
    return body;
  } catch (error) {
    signal?.throwIfAborted();
    if (error instanceof DOMException && error.name === 'AbortError') throw error;
    throw failure('invalid_response');
  }
}
export async function nativeJSON(path: string, options: NativeOptions = {}) {
  const controller = new AbortController();
  const signal = AbortSignal.any([
    controller.signal,
    ...(options.signal ? [options.signal] : []),
    ...(['/v1/ui/session', '/v1/ui/login'].includes(path) ? [] : [requestAccessSignal()]),
  ]);
  const timer =
    options.readOnly || !options.method || options.method === 'GET'
      ? setTimeout(() => controller.abort(new DOMException('Read deadline', 'TimeoutError')), 15000)
      : undefined;
  try {
    const response = await nativeRequest(path, { ...options, signal });
    return { response, body: await readNativeJSON(response, signal) };
  } catch (error) {
    if (controller.signal.aborted && !options.signal?.aborted) throw failure('read_timeout');
    throw error;
  } finally {
    if (timer) clearTimeout(timer);
  }
}
