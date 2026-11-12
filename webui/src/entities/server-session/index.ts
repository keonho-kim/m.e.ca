// Purpose: Expose the native server session lifecycle to higher FSD layers.
// Pattern: FSD entity public API.
// Usage: App and features import session state through this entry.
// Related: webui/src/entities/server-session/model/session.ts.

export {
  dismissNotice,
  establishSession,
  reportError,
  setServerLanguage,
  useServerSession,
} from '@/entities/server-session/model/session';
