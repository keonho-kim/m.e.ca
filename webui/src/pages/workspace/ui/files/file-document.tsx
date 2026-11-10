// Purpose: Open files directly for revision-bound editing and preview their current unsaved draft.
// Pattern: Read/edit/save state machine with bounded read-only fallback.
// Usage: Files owns one document; selecting a different file releases its editor and requests.
// Related: webui/src/pages/workspace/ui/files/file-header.tsx, webui/src/shared/ui/code-editor/index.tsx, internal/fileview/editor.go.

import { AlertDialog } from '@astryxdesign/core/AlertDialog';
import { Banner } from '@astryxdesign/core/Banner';
import { Button } from '@astryxdesign/core/Button';
import { useClickableContainer } from '@astryxdesign/core/hooks';
import { HStack, Layout, LayoutContent, VStack } from '@astryxdesign/core/Layout';
import { Spinner } from '@astryxdesign/core/Spinner';
import { Text } from '@astryxdesign/core/Text';
import { useQuery } from '@tanstack/react-query';
import { lazy, Suspense, useCallback, useEffect, useRef, useState } from 'react';
import { reportError } from '@/entities/server-session';
import { type Target, useWorkspaceStore } from '@/pages/workspace/model/store';
import { FileHeader } from '@/pages/workspace/ui/files/file-header';
import { call, type EditorDocumentJson } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { codeLanguage } from '@/shared/lib/code-language';
import { describeError, failure } from '@/shared/lib/errors';
import type { EditorHandle } from '@/shared/ui/code-editor';
import { CodePreview } from '@/shared/ui/code-preview';
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
  const editor = useRef<EditorHandle>(null),
    content = useRef<HTMLDivElement>(null);
  const mounted = useRef(true),
    focusPending = useRef(false);
  const reading = useRef<AbortController | null>(null),
    saving = useRef<AbortController | null>(null);
  const dirty = useWorkspaceStore((s) => s.fileDirty);
  const [doc, setDoc] = useState<EditorDocumentJson | null>(null);
  const [busy, setBusy] = useState(true),
    [readonly, setReadonly] = useState(false);
  const [problem, setProblem] = useState(''),
    [reason, setReason] = useState('');
  const [raw, setRaw] = useState(true),
    [version, setVersion] = useState(0);
  const [offsets, setOffsets] = useState([0]),
    [revision, setRevision] = useState('');
  const [editorReady, setEditorReady] = useState(false);
  const [draftPreview, setDraftPreview] = useState<string | undefined>();
  const [changed, setChanged] = useState(false),
    observed = useRef(externalRevision);
  const [discard, setDiscard] = useState(false);
  const offset = offsets.at(-1) ?? 0;
  const preview = useQuery({
    queryKey: ['preview', target.workspace, path, offset, revision],
    queryFn: async ({ signal }) => {
      const reply = await call(
        target.workspace,
        {
          previewFile: { path, offset, limit: 65536, ...(offset ? { revision } : {}) },
        },
        signal
      );
      if (!reply.filePreview) throw failure('invalid_response');
      return reply.filePreview;
    },
    enabled: readonly,
    staleTime: 0,
    gcTime: 0,
  });
  const readDocument = useCallback(async () => {
    reading.current?.abort();
    const controller = new AbortController();
    reading.current = controller;
    setBusy(true);
    setDoc(null);
    setReadonly(false);
    setProblem('');
    setReason('');
    setEditorReady(false);
    setDraftPreview(undefined);
    setRaw(true);
    setOffsets([0]);
    setRevision('');
    useWorkspaceStore.getState().set({ fileDirty: false });
    try {
      const reply = await call(target.workspace, { editorRead: { path } }, controller.signal);
      if (!mounted.current || controller.signal.aborted) return;
      if (!reply.editor) throw failure('invalid_response');
      if (reply.editor.editable) {
        setDoc(reply.editor);
        setVersion((value) => value + 1);
        focusPending.current = true;
      } else {
        setReadonly(true);
        setReason(reply.editor.reason ?? '');
      }
      setChanged(false);
    } catch (error) {
      if (mounted.current && !controller.signal.aborted) {
        setProblem(describeError(error));
        reportError(error);
      }
    } finally {
      if (mounted.current && reading.current === controller) {
        reading.current = null;
        setBusy(false);
      }
    }
  }, [target.workspace, path]);
  useEffect(() => {
    mounted.current = true;
    void readDocument();
    return () => {
      mounted.current = false;
      reading.current?.abort();
      saving.current?.abort();
      useWorkspaceStore.getState().set({ fileDirty: false });
    };
  }, [readDocument]);
  useEffect(() => {
    if (externalRevision !== observed.current) {
      observed.current = externalRevision;
      setChanged(true);
    }
  }, [externalRevision]);
  useEffect(() => {
    if (raw && editorReady && focusPending.current) {
      editor.current?.focus();
      focusPending.current = false;
    }
  }, [raw, editorReady]);
  const save = async () => {
    if (!doc || !editor.current || busy || saving.current) return;
    const controller = new AbortController();
    saving.current = controller;
    setBusy(true);
    const captured = editor.current.capture();
    try {
      const reply = await call(
        target.workspace,
        {
          editorSave: { path, text: captured.text, revision: doc.revision },
        },
        controller.signal
      );
      if (!mounted.current || controller.signal.aborted) return;
      if (!reply.editor?.revision) throw failure('invalid_response');
      setDoc((value) => (value ? { ...value, revision: reply.editor?.revision } : value));
      captured.saved();
      setChanged(false);
    } catch (error) {
      if (mounted.current && !controller.signal.aborted) reportError(error);
    } finally {
      if (mounted.current && saving.current === controller) {
        saving.current = null;
        setBusy(false);
      }
    }
  };
  const requestReset = () => {
    if (busy) return;
    if (useWorkspaceStore.getState().fileDirty) setDiscard(true);
    else void readDocument();
  };
  const isMarkdown = /\.(md|markdown)$/i.test(path);
  const mode = isMarkdown && !raw ? 'preview' : 'raw';
  const changeMode = (value: string) => {
    if (busy) return;
    if (value === 'preview' && isMarkdown) {
      if (doc) {
        if (!editor.current || !editorReady) return;
        setDraftPreview(editor.current.capture().text);
      }
      setRaw(false);
    } else if (value === 'raw') {
      focusPending.current = true;
      setDraftPreview(undefined);
      setRaw(true);
    }
  };
  const previewClick = useClickableContainer({
    containerRef: content,
    disabled: !doc || !editorReady || mode !== 'preview' || busy,
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
            ready={editorReady}
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
                    initialText={doc.text ?? ''}
                    onReady={() => {
                      setDoc((value) => (value ? { ...value, text: undefined } : value));
                      setEditorReady(true);
                    }}
                    onDirty={(fileDirty) => useWorkspaceStore.getState().set({ fileDirty })}
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
                    endContent={<Button label={t('retry')} onClick={() => readDocument()} />}
                  />
                )}
                {readonly && reason && (
                  <Text type="supporting" color="secondary">
                    {reason}
                  </Text>
                )}
                {readonly && preview.isPending && <Spinner label={t('loading')} />}
                {readonly && preview.error && (
                  <Banner
                    status="error"
                    title={describeError(preview.error)}
                    endContent={<Button label={t('retry')} onClick={() => preview.refetch()} />}
                  />
                )}
                {readonly &&
                  preview.data &&
                  (mode === 'preview' ? (
                    <MarkdownText text={preview.data.text ?? ''} />
                  ) : (
                    <CodePreview code={preview.data.text ?? ''} language={codeLanguage(path)} />
                  ))}
              </VStack>
            )}
          </LayoutContent>
        }
        footer={
          readonly ? (
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
        isOpen={discard}
        onOpenChange={setDiscard}
        title={t('dirty')}
        description={t('unsavedFile')}
        cancelLabel={t('keepEditing')}
        actionLabel={t('discard')}
        onAction={() => {
          setDiscard(false);
          void readDocument();
        }}
      />
    </>
  );
}
