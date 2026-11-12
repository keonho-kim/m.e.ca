// Purpose: Establish an authenticated browser session.
// Pattern: Authentication use cases.
// Usage: ConnectPage submits the explicit token handoff.
// Related: webui/src/features/server-access/api/access.ts.

import { establishSession } from '@/entities/server-session';
import { login } from '@/features/server-access/api/access';
export async function authenticate(token: string) {
  await login(token);
  establishSession();
}
