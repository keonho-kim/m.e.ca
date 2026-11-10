// Purpose: Establish or explicitly end an authenticated browser session.
// Pattern: Authentication use cases.
// Usage: ConnectPage and workspace Settings share these actions.
// Related: webui/src/features/server-access/api/access.ts.

import { endSession, establishSession } from '@/entities/server-session';
import { login, logout } from '@/features/server-access/api/access';
export async function authenticate(token: string) {
  await login(token);
  establishSession();
}
export async function disconnect() {
  await logout();
  endSession();
}
