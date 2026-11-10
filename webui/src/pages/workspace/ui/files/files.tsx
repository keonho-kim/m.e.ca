// Purpose: Show one current-directory list to the right of the active file document.
// Pattern: Master/detail navigator with a cancellable directory observer.
// Usage: Header, settings shortcut and !dir open this surface.
// Related: webui/src/pages/workspace/ui/files/directory-column.tsx, webui/src/pages/workspace/ui/files/file-document.tsx.

import { AlertDialog } from '@astryxdesign/core/AlertDialog';
import { Banner } from '@astryxdesign/core/Banner';
import { EmptyState } from '@astryxdesign/core/EmptyState';
import { useMediaQuery } from '@astryxdesign/core/hooks';
import { Layout, LayoutContent, LayoutPanel } from '@astryxdesign/core/Layout';
import { useEffect, useState } from 'react';
import { reportError, useServerSession } from '@/entities/server-session';
import { type Target, useWorkspaceStore } from '@/pages/workspace/model/store';
import { DirectoryColumn } from '@/pages/workspace/ui/files/directory-column';
import { FileDocument } from '@/pages/workspace/ui/files/file-document';
import { queries } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
export default function Files({ target }: { target: Target }) {
  const t = useLocale(),
    folder = useWorkspaceStore((s) => s.folder);
  const connected = useServerSession((s) => s.connected);
  const [columns, setColumns] = useState([folder]),
    [file, setFile] = useState('');
  const [revision, setRevision] = useState(0),
    [watchError, setWatchError] = useState('');
  const [watchEpoch, setWatchEpoch] = useState(0);
  const current = columns.at(-1) ?? '.';
  const watchKey = `${target.workspace}/${current}/${watchEpoch}`;
  const [watched, setWatched] = useState('');
  const compact = useMediaQuery('(max-width: 1024px)');
  const [pending, setPending] = useState<{ run: () => void } | null>(null);
  const navigate = (run: () => void) => {
    if (useWorkspaceStore.getState().fileDirty) setPending({ run });
    else run();
  };
  useEffect(() => {
    setColumns([folder]);
    setFile('');
  }, [folder]);
  useEffect(() => {
    if (!connected) return;
    setWatchError('');
    let first = true;
    const source = new EventSource(
      `/v1/files/events?workspace_id=${encodeURIComponent(target.workspace)}&path=${encodeURIComponent(current)}`
    );
    source.addEventListener('change', (raw) => {
      try {
        const event = JSON.parse(raw.data) as {
          refresh?: boolean;
          error?: string;
        };
        if (event.error) {
          setWatchError(event.error);
          setWatched(watchKey);
          source.close();
          return;
        }
        if (event.refresh) {
          if (!first || queries.getQueryState(['directory', target.workspace, current])?.data) {
            void queries.resetQueries({
              queryKey: ['directory', target.workspace, current],
              exact: true,
            });
          }
          setWatched(watchKey);
          if (!first) setRevision((value) => value + 1);
        }
        first = false;
      } catch (error) {
        reportError(error);
        source.close();
      }
    });
    source.onerror = () => {
      setWatchError(t('fileWatchReconnecting'));
      setWatched(watchKey);
    };
    source.onopen = () => setWatchError('');
    return () => source.close();
  }, [current, target.workspace, t, connected, watchKey]);
  const open = (path: string, directory: boolean) => {
    useWorkspaceStore.getState().set({ fileDirty: false });
    if (directory) {
      setColumns([...columns, path]);
      setFile('');
    } else setFile(path);
  };
  const listing = (
    <DirectoryColumn
      path={current}
      enabled={watched === watchKey}
      target={target}
      parent={
        columns.length > 1
          ? () =>
              navigate(() => {
                setColumns(columns.slice(0, -1));
                setFile('');
              })
          : undefined
      }
      refresh={() => {
        setWatchEpoch((value) => value + 1);
        setRevision((value) => value + 1);
      }}
      open={(next, directory) => {
        if (!directory && next === file) return;
        navigate(() => open(next, directory));
      }}
    />
  );
  return (
    <>
      <Layout
        height="fill"
        padding={0}
        className="files-workspace"
        content={
          <LayoutContent padding={0} isScrollable={false} className="min-w-0">
            {watchError && <Banner status="warning" title={watchError} />}
            {file ? (
              <FileDocument
                key={`${target.workspace}/${file}`}
                target={target}
                path={file}
                externalRevision={revision}
                showFiles={compact ? () => navigate(() => setFile('')) : undefined}
              />
            ) : compact ? (
              listing
            ) : (
              <EmptyState title={t('fileUnavailable')} />
            )}
          </LayoutContent>
        }
        end={
          !compact ? (
            <LayoutPanel width={260} padding={0} isScrollable={false}>
              {listing}
            </LayoutPanel>
          ) : undefined
        }
      />
      <AlertDialog
        isOpen={!!pending}
        title={t('dirty')}
        description={t('unsavedFile')}
        onOpenChange={(open) => {
          if (!open) setPending(null);
        }}
        cancelLabel={t('keepEditing')}
        actionLabel={t('discard')}
        onAction={() => {
          pending?.run();
          setPending(null);
        }}
      />
    </>
  );
}
