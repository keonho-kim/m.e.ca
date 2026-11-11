// Purpose: Observe one directory and revalidate only its affected open document.
// Pattern: Hook-owned SSE lifecycle with bounded filename notifications.
// Usage: Files owns navigation; this hook owns watcher readiness, errors and dirty hints.
// Related: webui/src/shared/api/event-stream.ts, internal/serverapi/files_watch.go.
import { useEffect, useRef, useState } from 'react';
import { useServerSession } from '@/entities/server-session';
import { queries, subscribeEventStream } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { ApiError, describeError } from '@/shared/lib/errors';

export function useFileEvents(workspace: string, folder: string, file: string, epoch: number) {
  const t = useLocale(),
    connected = useServerSession((state) => state.connected);
  const document = useRef(file);
  document.current = file;
  const key = JSON.stringify([workspace, folder, epoch]);
  const [ready, setReady] = useState(''),
    [error, setError] = useState(''),
    [revision, setRevision] = useState(0);
  useEffect(() => {
    if (!connected) return;
    let first = true;
    return subscribeEventStream({
      connect: async () => ({
        url: `/v1/files/events?workspace_id=${encodeURIComponent(workspace)}&path=${encodeURIComponent(folder)}`,
        events: {
          change: (raw) => {
            const event = JSON.parse(raw) as {
              refresh?: boolean;
              error?: string;
              errorCode?: string;
              changedNames?: string[];
              rescan?: boolean;
            };
            if (event.error)
              throw new ApiError(event.errorCode ?? 'unavailable', event.error, true);
            setReady(key);
            if (!event.refresh) return;
            if (!first || queries.getQueryState(['directory', workspace, folder])?.data)
              void queries.resetQueries({
                queryKey: ['directory', workspace, folder],
                exact: true,
              });
            const open = document.current.split('/').at(-1);
            if (open && (event.rescan || event.changedNames?.includes(open)))
              setRevision((value) => value + 1);
            first = false;
          },
        },
      }),
      status: (value) => {
        setError(value === 'reconnecting' ? t('fileWatchReconnecting') : '');
      },
      failed: (value) => {
        setReady(key);
        setError(describeError(value));
        return (
          !(value instanceof ApiError) ||
          ['network', 'unavailable', 'resource_exhausted', 'invalid_response'].includes(value.code)
        );
      },
    });
  }, [workspace, folder, connected, key, t]);
  return { ready: ready === key, error, revision };
}
