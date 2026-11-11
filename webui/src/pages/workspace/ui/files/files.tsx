// Purpose: Show one current-directory list to the right of the active file document.
// Pattern: Master/detail navigator with a cancellable directory observer.
// Usage: Header, settings shortcut and !dir open this surface.
// Related: webui/src/pages/workspace/ui/files/directory-column.tsx, webui/src/pages/workspace/ui/files/file-document.tsx.

import { AlertDialog } from '@astryxdesign/core/AlertDialog';
import { Banner } from '@astryxdesign/core/Banner';
import { EmptyState } from '@astryxdesign/core/EmptyState';
import { useMediaQuery } from '@astryxdesign/core/hooks';
import { Layout, LayoutContent, LayoutPanel } from '@astryxdesign/core/Layout';
import { useEffect, useRef, useState } from 'react';
import { reportError } from '@/entities/server-session';
import { useFileEvents } from '@/pages/workspace/model/file-events';
import { mutateFile } from '@/pages/workspace/model/file-mutations';
import { type Target, useWorkspaceStore } from '@/pages/workspace/model/store';
import { DirectoryColumn } from '@/pages/workspace/ui/files/directory-column';
import { FileDocument } from '@/pages/workspace/ui/files/file-document';
import { type FileMutationJson, queries } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { ApiError } from '@/shared/lib/errors';
export default function Files({ target }: { target: Target }) {
  const t = useLocale(),
    folder = useWorkspaceStore((s) => s.folder);
  const [columns, setColumns] = useState([folder]),
    [file, setFile] = useState('');
  const [watchEpoch, setWatchEpoch] = useState(0);
  const current = columns.at(-1) ?? '.';
  const watcher = useFileEvents(target.workspace, current, file, watchEpoch);
  const watchError = watcher.error;
  const compact = useMediaQuery('(max-width: 1024px)');
  const [pending, setPending] = useState<{ run: () => void } | null>(null);
  const [mutating, setMutating] = useState(false);
  const mutation = useRef(false);
  const navigate = (run: () => void) => {
    if (useWorkspaceStore.getState().fileDirty) setPending({ run });
    else run();
  };
  useEffect(() => {
    setColumns([folder]);
    setFile('');
  }, [folder]);
  const open = (path: string, directory: boolean) => {
    useWorkspaceStore.getState().set({ fileDirty: false });
    if (directory) {
      setColumns([...columns, path]);
      setFile('');
    } else setFile(path);
  };
  const mutate = (change: FileMutationJson) => {
    const source = change.path;
    if (!source || mutation.current) return;
    const affectsFile = file === source || file.startsWith(`${source}/`);
    const run = async () => {
      if (mutation.current) return;
      mutation.current = true;
      setMutating(true);
      try {
        const result = await mutateFile(target, change);
        const destination = result.path;
        if (affectsFile) {
          useWorkspaceStore.getState().set({ fileDirty: false });
          setFile(destination ? destination + file.slice(source.length) : '');
        }
        await queries.invalidateQueries({
          queryKey: ['directory', target.workspace],
          refetchType: 'none',
        });
        if (result.errorCode)
          reportError(new ApiError(result.errorCode, result.errorMessage ?? ''));
        if (watchError)
          void queries.resetQueries({
            queryKey: ['directory', target.workspace, current],
            exact: true,
          });
      } catch (error) {
        reportError(error);
      } finally {
        mutation.current = false;
        setMutating(false);
      }
    };
    if (affectsFile)
      navigate(() => {
        void run();
      });
    else void run();
  };
  const listing = (
    <DirectoryColumn
      path={current}
      enabled={watcher.ready}
      target={target}
      busy={mutating}
      mutate={mutate}
      parentPath={columns.at(-2)}
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
      }}
      open={(next, directory) => {
        if (mutation.current) return;
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
        className="files-workspace meca-pane-enter"
        content={
          <LayoutContent padding={0} isScrollable={false} className="min-w-0" inert={mutating}>
            {watchError && <Banner status="warning" title={watchError} />}
            {file ? (
              <FileDocument
                key={`${target.workspace}/${file}`}
                target={target}
                path={file}
                externalRevision={watcher.revision}
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
