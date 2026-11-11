// Purpose: Present an open document's editing, preview and discard controls.
// Pattern: Astryx document view over independent I/O and paging owners.
// Usage: Files mounts one document; its editor retains text and undo across display modes.
// Related: webui/src/pages/workspace/model/file-document.ts, webui/src/pages/workspace/ui/files/file-preview.tsx.
import { AlertDialog } from '@astryxdesign/core/AlertDialog';
import { Banner } from '@astryxdesign/core/Banner';
import { Button } from '@astryxdesign/core/Button';
import { useClickableContainer } from '@astryxdesign/core/hooks';
import { Layout, LayoutContent, VStack } from '@astryxdesign/core/Layout';
import { Spinner } from '@astryxdesign/core/Spinner';
import { Text } from '@astryxdesign/core/Text';
import { lazy, Suspense, useEffect, useRef, useState } from 'react';
import { reportError } from '@/entities/server-session';
import { useDocument } from '@/pages/workspace/model/file-document';
import { useFilePreview } from '@/pages/workspace/model/file-preview';
import { type Target, useWorkspaceStore } from '@/pages/workspace/model/store';
import { FileHeader } from '@/pages/workspace/ui/files/file-header';
import { FilePages, FilePreview } from '@/pages/workspace/ui/files/file-preview';
import { useLocale } from '@/shared/i18n';
import { MarkdownText } from '@/shared/ui/markdown';

const Editor = lazy(() => import('@/shared/ui/code-editor'));
export function FileDocument({
  target,
  path,
  externalRevision,
  showFiles,
}: {
  target: Target;
  path: string;
  externalRevision: number;
  showFiles?: () => void;
}) {
  const t = useLocale();
  const content = useRef<HTMLDivElement>(null);
  const state = useDocument(target, path, externalRevision);
  const {
    editor,
    doc,
    maxBytes,
    busy,
    readonly,
    problem,
    reason,
    version,
    ready,
    changed,
    read,
    save,
  } = state;
  const preview = useFilePreview(target.workspace, path, readonly, version);
  const dirty = useWorkspaceStore((s) => s.fileDirty);
  const [raw, setRaw] = useState(true),
    [discard, setDiscard] = useState(false);
  const [draftPreview, setDraftPreview] = useState<string | undefined>();
  useEffect(() => {
    setRaw(true);
    setDraftPreview(undefined);
  }, [version]);
  useEffect(() => {
    if (raw && ready) editor.current?.focus();
  }, [raw, ready, editor]);
  const requestReset = () => {
    if (busy) return;
    if (useWorkspaceStore.getState().fileDirty) setDiscard(true);
    else void read();
  };
  const isMarkdown = /\.(md|markdown)$/i.test(path);
  const mode = isMarkdown && !raw ? 'preview' : 'raw';
  const changeMode = (value: string) => {
    if (busy) return;
    if (value === 'preview' && isMarkdown) {
      if (doc) {
        if (!editor.current || !ready) return;
        setDraftPreview(editor.current.text());
      }
      setRaw(false);
    } else if (value === 'raw') {
      setDraftPreview(undefined);
      setRaw(true);
    }
  };
  const previewClick = useClickableContainer({
    containerRef: content,
    disabled: !doc || !ready || mode !== 'preview' || busy,
    onClick: () => changeMode('raw'),
  });
  return (
    <>
      <Layout
        height="fill"
        padding={0}
        className="file-document min-w-0 max-w-full"
        header={
          <FileHeader
            path={path}
            editing={!!doc}
            readonly={readonly}
            showFiles={showFiles}
            dirty={dirty}
            busy={busy}
            ready={ready}
            markdown={isMarkdown}
            mode={mode}
            changeMode={changeMode}
            save={save}
            cancel={requestReset}
          />
        }
        content={
          <LayoutContent
            ref={content}
            {...previewClick}
            padding={doc && mode === 'raw' ? 0 : 5}
            isScrollable={!doc || mode === 'preview'}
            className={`min-w-0 ${doc && mode === 'preview' ? 'cursor-text' : ''}`}
          >
            {changed && (
              <Banner
                status="warning"
                title={t('externalChange')}
                endContent={
                  <Button label={t('refresh')} isDisabled={busy} onClick={requestReset} />
                }
              />
            )}
            {doc && (
              <VStack hidden={mode !== 'raw'} height="100%" gap={0} className="min-h-0 min-w-0">
                <Suspense fallback={<Spinner label={t('loading')} />}>
                  <Editor
                    key={`${path}:${version}`}
                    ref={editor}
                    path={path}
                    maxBytes={maxBytes}
                    initialText={doc.text ?? ''}
                    visible={mode === 'raw'}
                    onError={reportError}
                    onReady={state.onReady}
                    onDirty={state.onDirty}
                  />
                </Suspense>
              </VStack>
            )}
            {doc && mode === 'preview' && <MarkdownText text={draftPreview ?? ''} />}
            {!doc && (
              <VStack gap={4} className="min-w-0">
                {busy && <Spinner label={t('loading')} />}
                {problem && (
                  <Banner
                    status="error"
                    title={problem}
                    endContent={<Button label={t('retry')} onClick={() => read()} />}
                  />
                )}
                {readonly && reason && (
                  <Text type="supporting" color="secondary">
                    {reason}
                  </Text>
                )}
                {readonly && <FilePreview state={preview} path={path} mode={mode} />}
              </VStack>
            )}
          </LayoutContent>
        }
        footer={readonly ? <FilePages state={preview} /> : undefined}
      />
      <AlertDialog
        isOpen={discard}
        onOpenChange={setDiscard}
        title={t('dirty')}
        description={t('unsavedFile')}
        cancelLabel={t('keepEditing')}
        actionLabel={t('discard')}
        onAction={() => {
          setDiscard(false);
          void read();
        }}
      />
    </>
  );
}
