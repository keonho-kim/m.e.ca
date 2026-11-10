// Purpose: Expose explicit native server authentication actions.
// Pattern: FSD feature public API.
// Usage: Connect and workspace pages use this feature.
// Related: webui/src/features/server-access/model/access.ts.

export { authenticate, disconnect } from '@/features/server-access/model/access';
