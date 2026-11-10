// Purpose: Authenticate before downloading the application workspace and its reading/editor dependencies.
// Pattern: Lazy browser composition root.
// Usage: TanStack Router mounts App for every embedded UI route.
// Related: webui/src/pages/workspace/ui/workspace-page.tsx, webui/src/pages/connect/ui/connect-page.tsx.

import { VStack } from '@astryxdesign/core/Layout';
import { Text } from '@astryxdesign/core/Text';
import { onlineManager } from '@tanstack/react-query';
import { lazy, Suspense, useEffect, useState } from 'react';
import { establishSession, useServerSession } from '@/entities/server-session';
import { Login } from '@/pages/connect';
import { call, queries } from '@/shared/api';
import { useLocale } from '@/shared/i18n';

const WorkspaceShell = lazy(() => import('@/pages/workspace'));
export function App() {
  const connected = useServerSession((s) => s.connected),
    t = useLocale();
  const [checking, setChecking] = useState(true);
  const [opened, setOpened] = useState<number | null>(null);
  const logoutRevision = useServerSession((s) => s.logoutRevision);
  useEffect(() => {
    onlineManager.setOnline(connected);
    if (connected) setOpened(logoutRevision);
  }, [connected, logoutRevision]);
  useEffect(() => {
    const controller = new AbortController();
    void call('', { capabilities: {} }, controller.signal)
      .then((reply) => {
        if (!controller.signal.aborted && reply.capabilities)
          queries.setQueryData(['capabilities'], reply.capabilities);
        if (!controller.signal.aborted) establishSession(reply.capabilities?.language ?? 'auto');
      })
      .catch(() => {})
      .finally(() => {
        if (!controller.signal.aborted) setChecking(false);
      });
    return () => controller.abort();
  }, []);
  const loading = (
    <VStack className="startup-screen" gap={4}>
      <Text className="wordmark">MECA</Text>
      <Text as="p">{t('connecting')}</Text>
    </VStack>
  );
  if (checking) return loading;
  return (
    <>
      {(connected || opened === logoutRevision) && (
        <VStack hidden={!connected} gap={4}>
          <Suspense fallback={loading}>
            <WorkspaceShell />
          </Suspense>
        </VStack>
      )}
      {!connected && <Login />}
    </>
  );
}
