// Purpose: Recover unavailable application modules without depending on the failed import.
// Pattern: Scoped React error boundary and explicit document reload.
// Usage: App and lazy feature boundaries preserve the surrounding composer and editor.
// Related: webui/src/app/main.tsx, webui/src/shared/lib/reload/index.ts.
import { Button } from '@astryxdesign/core/Button';
import { Center } from '@astryxdesign/core/Center';
import { EmptyState } from '@astryxdesign/core/EmptyState';
import { HStack } from '@astryxdesign/core/Layout';
import { Component, type ReactNode, useEffect, useState, useSyncExternalStore } from 'react';
import { nativeJSON } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { describeError } from '@/shared/lib/errors';
import {
  discardRecoveryFile,
  downloadRecoveryFile,
  observeRecoveryFile,
  recoveryFile,
} from '@/shared/lib/reload';

declare const __MECA_UI_BUILD__: string;

export function Recovery() {
  const draft = useSyncExternalStore(observeRecoveryFile, recoveryFile);
  const t = useLocale(),
    [changed, setChanged] = useState(false),
    [error, setError] = useState(''),
    [busy, setBusy] = useState(false);
  const probe = async (signal?: AbortSignal) => {
    const { response, body } = await nativeJSON('/v1/ui/session', { signal });
    if (!response.ok) return false;
    if (body && typeof body === 'object' && 'buildId' in body)
      setChanged(typeof __MECA_UI_BUILD__ !== 'undefined' && body.buildId !== __MECA_UI_BUILD__);
    return true;
  };
  useEffect(() => {
    const controller = new AbortController();
    void probe(controller.signal).catch(() => {});
    return () => controller.abort();
  }, []);
  return (
    <Center padding={6}>
      <EmptyState
        title="MECA"
        description={error || t(changed ? 'updatedApp' : 'failedApp')}
        actions={
          <HStack gap={3} wrap="wrap">
            {draft && <Button label={t('downloadDraft')} onClick={downloadRecoveryFile} />}
            <Button
              label={t(draft ? 'discardReload' : 'reloadApp')}
              isLoading={busy}
              onClick={async () => {
                setBusy(true);
                try {
                  if (await probe()) {
                    if (draft) discardRecoveryFile();
                    window.location.reload();
                  }
                } catch (error) {
                  setError(describeError(error));
                } finally {
                  setBusy(false);
                }
              }}
            />
          </HStack>
        }
      />
    </Center>
  );
}

export class FeatureBoundary extends Component<{ children: ReactNode }, { failed: boolean }> {
  state = { failed: false };
  static getDerivedStateFromError() {
    return { failed: true };
  }
  render() {
    return this.state.failed ? <Recovery /> : this.props.children;
  }
}
