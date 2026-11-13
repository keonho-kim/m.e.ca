// Purpose: Own native server authentication lifetime, language and safe failure feedback.
// Pattern: Narrow Zustand session store.
// Usage: App startup, login and workspace actions share this session.
// Related: webui/src/shared/lib/errors/index.ts.

import { create } from 'zustand';
import { queries, requestAccessAllowed, setRequestAccess } from '@/shared/api';
import { ApiError, describeError, isCancelled } from '@/shared/lib/errors';

interface ServerSession {
  connected: boolean;
  language: string;
  notice: string;
}
export const useServerSession = create<ServerSession>(() => ({
  connected: false,
  language: 'auto',
  notice: '',
}));
export function establishSession(language?: string) {
  const restoring = !requestAccessAllowed();
  setRequestAccess(true);
  useServerSession.setState({ connected: true, notice: '', ...(language ? { language } : {}) });
  if (restoring) void queries.resetQueries();
}
export function setServerLanguage(language: string) {
  useServerSession.setState({ language });
}
export function dismissNotice() {
  useServerSession.setState({ notice: '' });
}
export function reportError(error: unknown) {
  if (isCancelled(error)) return;
  if (error instanceof ApiError && error.code === 'unauthorized' && requestAccessAllowed()) {
    setRequestAccess(false);
    void queries.cancelQueries();
    queries.clear();
  }
  useServerSession.setState({
    notice: describeError(error),
    ...(error instanceof ApiError && error.code === 'unauthorized' ? { connected: false } : {}),
  });
}
