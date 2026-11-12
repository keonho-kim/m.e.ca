// Purpose: Authenticate before downloading the application workspace and its reading/editor dependencies.
// Pattern: Lazy browser composition root.
// Usage: TanStack Router mounts App for every embedded UI route.
// Related: webui/src/pages/workspace/ui/workspace-page.tsx, webui/src/pages/connect/ui/connection-loading.tsx.

import { VStack } from '@astryxdesign/core/Layout';
import { onlineManager } from '@tanstack/react-query';
import { lazy, Suspense, useEffect, useState } from 'react';
import { establishSession, useServerSession } from '@/entities/server-session';
import { hasBrowserSession } from '@/features/server-access';
import { ConnectionLoading, Login } from '@/pages/connect';
import { call, queries } from '@/shared/api';

const WorkspaceShell = lazy(() => import('@/pages/workspace'));
export function App() {
  const connected = useServerSession((s) => s.connected);
  const [checking, setChecking] = useState(true);
  const [opened, setOpened] = useState(false);
  useEffect(() => {
    onlineManager.setOnline(connected);
    if (connected) setOpened(true);
  }, [connected]);
  useEffect(() => {
    const controller = new AbortController();
    void hasBrowserSession(controller.signal)
      .then((authenticated) =>
        authenticated ? call('', { capabilities: {} }, controller.signal) : null
      )
      .then((reply) => {
        if (!reply) return;
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
  const loading = <ConnectionLoading checking={checking} />;
  if (checking) return loading;
  return (
    <>
      {(connected || opened) && (
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
