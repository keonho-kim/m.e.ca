// Purpose: Authenticate and revoke the native server cookie without storing bearer tokens.
// Pattern: Same-origin authentication adapter.
// Usage: Server-access actions call these endpoints.
// Related: internal/serverapi/ui_auth.go.

import { nativeJSON, nativeRequest } from '@/shared/api';
import { failure } from '@/shared/lib/errors';
export async function hasBrowserSession(signal?: AbortSignal): Promise<boolean> {
  const { response, body: value } = await nativeJSON('/v1/ui/session', { signal });
  if (!response.ok) throw failure('network');
  if (
    typeof value !== 'object' ||
    value === null ||
    !('authenticated' in value) ||
    typeof value.authenticated !== 'boolean'
  )
    throw failure('invalid_response');
  return value.authenticated;
}

export async function login(token: string) {
  const value = token
    .trim()
    .replace(/^Server token:\s*/, '')
    .trim();
  const response = await nativeRequest('/v1/ui/login', {
    method: 'POST',
    headers: { Authorization: `Bearer ${value}` },
  });
  await response.body?.cancel();
  if (!response.ok) throw failure(response.status === 403 ? 'forbidden' : 'network');
  if (!(await hasBrowserSession())) throw failure('browser_cookie');
}
