// Purpose: Authenticate and revoke the native server cookie without storing bearer tokens.
// Pattern: Same-origin authentication adapter.
// Usage: Server-access actions call these endpoints.
// Related: internal/serverapi/ui_auth.go.

import { failure } from '@/shared/lib/errors';
export async function login(token: string) {
  const response = await fetch('/v1/ui/login', {
    method: 'POST',
    headers: { Authorization: 'Bearer ' + token, 'X-Meca-UI': '1' },
    credentials: 'same-origin',
  });
  if (!response.ok) throw failure('unauthorized');
}
export async function logout() {
  const response = await fetch('/v1/ui/logout', {
    method: 'POST',
    headers: { 'X-Meca-UI': '1' },
    credentials: 'same-origin',
  });
  if (!response.ok) throw failure('network');
}
