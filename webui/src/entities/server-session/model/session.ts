// Purpose: Own native server authentication lifetime, language and safe failure feedback.
// Pattern: Narrow Zustand session store.
// Usage: App startup, login and workspace actions share this session.
// Related: webui/src/shared/lib/errors/index.ts.

import { create } from 'zustand';
import { ApiError, describeError } from '@/shared/lib/errors';

interface ServerSession {
  connected: boolean;
  logoutRevision: number;
  language: string;
  notice: string;
}
export const useServerSession = create<ServerSession>(() => ({
  connected: false,
  logoutRevision: 0,
  language: 'auto',
  notice: '',
}));
export function establishSession(language?: string) {
  useServerSession.setState({ connected: true, notice: '', ...(language ? { language } : {}) });
}
export function endSession() {
  useServerSession.setState((state) => ({
    connected: false,
    logoutRevision: state.logoutRevision + 1,
    notice: '',
  }));
}
export function setServerLanguage(language: string) {
  useServerSession.setState({ language });
}
export function dismissNotice() {
  useServerSession.setState({ notice: '' });
}
export function reportError(error: unknown) {
  useServerSession.setState({
    notice: describeError(error),
    ...(error instanceof ApiError && error.code === 'unauthorized' ? { connected: false } : {}),
  });
}
