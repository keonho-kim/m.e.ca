// Purpose: Preview bounded text pages or explicitly edit a complete revision-bound document.
// Pattern: Read/edit/save state machine.
// Usage: Files owns one FileDocument and releases its editor on navigation.
// Related: webui/src/shared/ui/code-editor/index.tsx, internal/fileview/editor.go.

import { AlertDialog } from '@astryxdesign/core/AlertDialog';
import { Banner } from '@astryxdesign/core/Banner';
import { Button } from '@astryxdesign/core/Button';
import { HStack, Layout, LayoutContent, VStack } from '@astryxdesign/core/Layout';
import { Spinner } from '@astryxdesign/core/Spinner';
import { Text } from '@astryxdesign/core/Text';
import { Toolbar } from '@astryxdesign/core/Toolbar';
import { useQuery } from '@tanstack/react-query';
import { lazy, Suspense, useEffect, useRef, useState } from 'react';
import { reportError, useServerSession } from '@/entities/server-session';
import { type Target, useWorkspaceStore } from '@/pages/workspace/model/store';
import type { EditorDocumentJson } from '@/shared/api';
import { call } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { codeLanguage } from '@/shared/lib/code-language';
import type { EditorHandle } from '@/shared/ui/code-editor';
import { CodePreview } from '@/shared/ui/code-preview';
import { MarkdownText } from '@/shared/ui/markdown';

const Editor = lazy(() => import('@/shared/ui/code-editor'));
export function FileDocument({
  target,
  path,
  externalRevision,
}: {
  target: Target;
  path: string;
  externalRevision: number;
}) {
  const t = useLocale(),
    editor = useRef<EditorHandle>(null),
    mounted = useRef(true);
  const reading = useRef<AbortController | null>(null);
  const dirty = useWorkspaceStore((s) => s.fileDirty);
  const [doc, setDoc] = useState<EditorDocumentJson | null>(null),
    [busy, setBusy] = useState(false);
  const [raw, setRaw] = useState(false),
    [offsets, setOffsets] = useState([0]),
    [revision, setRevision] = useState('');
  const [changed, setChanged] = useState(false),
    observed = useRef(externalRevision);
  const offset = offsets.at(-1) ?? 0;
  const preview = useQuery({
    queryKey: ['preview', target.workspace, path, offset, revision],
    queryFn: async ({ signal }) =>
      (
        await call(
          target.workspace,
          { previewFile: { path, offset, limit: 65536, ...(offset ? { revision } : {}) } },
          signal
        )
      ).filePreview!,
    enabled: !doc,
    gcTime: 0,
  });
  useEffect(() => {
    if (externalRevision !== observed.current) {
      observed.current = externalRevision;
      setChanged(true);
    }
  }, [externalRevision]);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      reading.current?.abort();
      useWorkspaceStore.getState().set({ fileDirty: false });
    };
  }, []);
  const openEditor = async () => {
    reading.current?.abort();
    const controller = new AbortController();
    reading.current = controller;
    setBusy(true);
    try {
      const reply = await call(target.workspace, { editorRead: { path } }, controller.signal);
      if (!mounted.current) return;
      if (!reply.editor?.editable) {
        useServerSession.setState({ notice: reply.editor?.reason ?? t('readonly') });
        return;
      }
      setDoc(reply.editor);
      setChanged(false);
    } catch (error) {
      if (!controller.signal.aborted) reportError(error);
    } finally {
      setBusy(false);
    }
  };
  const save = async () => {
    if (!doc || !editor.current) return;
    setBusy(true);
    const captured = editor.current.capture();
    try {
      const reply = await call(target.workspace, {
        editorSave: { path, text: captured.text, revision: doc.revision },
      });
      if (!mounted.current) return;
      if (reply.editor) setDoc((value) => ({ ...value, revision: reply.editor?.revision }));
      captured.saved();
      setChanged(false);
    } catch (error) {
      reportError(error);
    } finally {
      setBusy(false);
    }
  };
  const [discard, setDiscard] = useState<'cancel' | 'reload' | null>(null);
  const reset = (reload: boolean) => {
    setDoc(null);
    useWorkspaceStore.getState().set({ fileDirty: false });
    setOffsets([0]);
    setRevision('');
    if (reload) setChanged(false);
    void preview.refetch();
  };
  const requestReset = (action: 'cancel' | 'reload') => {
    if (busy) return;
    if (useWorkspaceStore.getState().fileDirty) setDiscard(action);
    else reset(action === 'reload');
  };
  const isMarkdown = /\.(md|markdown)$/i.test(path);
  return (
    <>
      <Layout
        height="fill"
        padding={0}
        className="file-document"
        header={
          <Toolbar
            className="document-toolbar"
            label={path}
            startContent={
              <Text type="label" maxLines={1} hasTruncateTooltip>
                {path}
              </Text>
            }
            endContent={
              <HStack gap={2}>
                {doc ? (
                  <>
                    <Button
                      label={t('cancel')}
                      isDisabled={busy}
                      onClick={() => requestReset('cancel')}
                    />
                    <Button
                      label={t('save')}
                      variant="primary"
                      isLoading={busy}
                      isDisabled={!dirty}
                      onClick={save}
                    />
                  </>
                ) : (
                  <>
                    <Button label={raw ? t('preview') : t('raw')} onClick={() => setRaw(!raw)} />
                    <Button label={t('editFile')} onClick={openEditor} isDisabled={busy} />
                  </>
                )}
              </HStack>
            }
          />
        }
        content={
          <LayoutContent padding={doc ? 0 : 5} isScrollable={!doc}>
            {changed && (
              <Banner
                status="warning"
                title={t('externalChange')}
                endContent={
                  <Button
                    label={t('refresh')}
                    isDisabled={busy}
                    onClick={() => requestReset('reload')}
                  />
                }
              />
            )}
            {doc ? (
              <Suspense fallback={<Spinner label={t('loading')} />}>
                <Editor
                  key={path}
                  ref={editor}
                  path={path}
                  initialText={doc.text ?? ''}
                  onReady={() => setDoc((value) => (value ? { ...value, text: undefined } : value))}
                  onDirty={(fileDirty) => useWorkspaceStore.getState().set({ fileDirty })}
                />
              </Suspense>
            ) : (
              <VStack gap={4}>
                {preview.isPending && <Spinner label={t('loading')} />}
                {preview.error && <Button label={t('retry')} onClick={() => preview.refetch()} />}
                {preview.data &&
                  (isMarkdown && !raw ? (
                    <MarkdownText text={preview.data.text ?? ''} />
                  ) : (
                    <CodePreview code={preview.data.text ?? ''} language={codeLanguage(path)} />
                  ))}
              </VStack>
            )}
          </LayoutContent>
        }
        footer={
          !doc ? (
            <HStack gap={2} padding={3} justify="end" wrap="wrap">
              <Text type="supporting" color="secondary">
                {offset} {t('bytes')}
              </Text>
              <Button
                label={t('parent')}
                isDisabled={offsets.length <= 1}
                onClick={() => setOffsets(offsets.slice(0, -1))}
              />
              <Button
                label={t('moreOriginal')}
                isDisabled={
                  !preview.data?.text ||
                  Number(preview.data.nextOffset ?? 0) >= Number(preview.data.totalBytes ?? 0)
                }
                onClick={() => {
                  setRevision(preview.data?.revision ?? '');
                  setOffsets([...offsets, Number(preview.data?.nextOffset ?? 0)]);
                }}
              />
            </HStack>
          ) : undefined
        }
      />
      <AlertDialog
        isOpen={!!discard}
        onOpenChange={(open) => {
          if (!open) setDiscard(null);
        }}
        title={t('dirty')}
        description={t('unsavedFile')}
        cancelLabel={t('keepEditing')}
        actionLabel={t('discard')}
        onAction={() => {
          reset(discard === 'reload');
          setDiscard(null);
        }}
      />
    </>
  );
}
