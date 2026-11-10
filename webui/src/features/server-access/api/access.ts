// Purpose: Authenticate and revoke the native server cookie without storing bearer tokens.
// Pattern: Same-origin authentication adapter.
// Usage: Server-access actions call these endpoints.
// Related: internal/serverapi/ui_auth.go.

import { failure } from '@/shared/lib/errors';
export async function hasBrowserSession(signal?: AbortSignal): Promise<boolean> {
  const response = await fetch('/v1/ui/session', { credentials: 'same-origin', signal });
  if (!response.ok) throw failure('network');
  const value: unknown = await response.json();
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
  const response = await fetch('/v1/ui/login', {
    method: 'POST',
    headers: { Authorization: `Bearer ${value}`, 'X-Meca-UI': '1' },
    credentials: 'same-origin',
  });
  if (!response.ok)
    throw failure(
      response.status === 401 ? 'unauthorized' : response.status === 403 ? 'forbidden' : 'network'
    );
  if (!(await hasBrowserSession())) throw failure('browser_cookie');
}
export async function logout() {
  const response = await fetch('/v1/ui/logout', {
    method: 'POST',
    headers: { 'X-Meca-UI': '1' },
    credentials: 'same-origin',
  });
  if (!response.ok) throw failure('network');
}
