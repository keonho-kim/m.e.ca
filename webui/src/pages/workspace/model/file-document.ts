// Purpose: Own a document's cancellable read/save lifecycle and revision acknowledgements.
// Pattern: Revision-bound document I/O hook.
// Usage: FileDocument owns display modes; its editor owns text and undo transactions.
// Related: webui/src/pages/workspace/api/queries.ts, webui/src/pages/workspace/ui/files/file-document.tsx.
import { useCallback, useEffect, useRef, useState } from 'react';
import { reportError } from '@/entities/server-session';
import { capabilitiesOptions } from '@/pages/workspace/api/queries';
import { type Target, useWorkspaceStore } from '@/pages/workspace/model/store';
import { call, type EditorDocumentJson, queries } from '@/shared/api';
import { describeError, failure } from '@/shared/lib/errors';
import type { EditorHandle } from '@/shared/ui/code-editor';

const markDirty = (fileDirty: boolean) => useWorkspaceStore.getState().set({ fileDirty });
export function useDocument(target: Target, path: string, externalRevision: number) {
  const editor = useRef<EditorHandle>(null),
    mounted = useRef(false);
  const reading = useRef<AbortController | null>(null),
    saving = useRef<AbortController | null>(null);
  const checking = useRef<AbortController | null>(null);
  const baseline = useRef<{ revision?: string; watchRevision?: string } | null>(null);
  const [doc, setDoc] = useState<EditorDocumentJson | null>(null);
  const [busy, setBusy] = useState(true),
    [readonly, setReadonly] = useState(false);
  const [problem, setProblem] = useState(''),
    [reason, setReason] = useState('');
  const [maxBytes, setMaxBytes] = useState(0),
    [version, setVersion] = useState(0);
  const [ready, setReady] = useState(false),
    [changed, setChanged] = useState(false);
  const observed = useRef(externalRevision);
  const read = useCallback(async () => {
    reading.current?.abort();
    const controller = new AbortController();
    reading.current = controller;
    setBusy(true);
    setDoc(null);
    baseline.current = null;
    setReadonly(false);
    setProblem('');
    setReason('');
    setReady(false);
    setVersion((value) => value + 1);
    markDirty(false);
    try {
      const capabilities = await queries.ensureQueryData(capabilitiesOptions);
      controller.signal.throwIfAborted();
      const limit = Number(capabilities.editorBytes);
      if (!Number.isSafeInteger(limit) || limit <= 0) throw failure('invalid_response');
      const reply = await call(target.workspace, { editorRead: { path } }, controller.signal);
      if (!mounted.current || controller.signal.aborted) return;
      if (!reply.editor) throw failure('invalid_response');
      setMaxBytes(limit);
      baseline.current = {
        revision: reply.editor.revision,
        watchRevision: reply.editor.watchRevision,
      };
      if (reply.editor.editable) {
        if (!reply.editor.revision) throw failure('invalid_response');
        setDoc(reply.editor);
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
    void read();
    return () => {
      mounted.current = false;
      reading.current?.abort();
      saving.current?.abort();
      checking.current?.abort();
      markDirty(false);
    };
  }, [read]);
  useEffect(() => {
    const original = baseline.current;
    if (externalRevision === observed.current || busy || !original) return;
    observed.current = externalRevision;
    checking.current?.abort();
    const controller = new AbortController();
    checking.current = controller;
    void call(
      target.workspace,
      { editorProbe: { path, watchRevision: original.watchRevision } },
      controller.signal
    )
      .then((reply) => {
        if (controller.signal.aborted || !mounted.current) return;
        const version = reply.editorVersion;
        if (!version) throw failure('invalid_response');
        const changed =
          !!version.missing ||
          (!version.unchanged && (!version.revision || version.revision !== original.revision));
        setChanged(changed);
        if (!changed && version.watchRevision && baseline.current === original)
          baseline.current = { ...original, watchRevision: version.watchRevision };
      })
      .catch((error) => {
        if (!controller.signal.aborted && mounted.current) reportError(error);
      });
    return () => controller.abort();
  }, [externalRevision, busy, doc, target.workspace, path]);
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
      baseline.current = {
        revision: reply.editor.revision,
        watchRevision: reply.editor.watchRevision,
      };
      setDoc((value) =>
        value
          ? {
              ...value,
              revision: reply.editor?.revision,
              watchRevision: reply.editor?.watchRevision,
            }
          : value
      );
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
  const onReady = useCallback(() => {
    setDoc((value) => (value ? { ...value, text: undefined } : value));
    setReady(true);
  }, []);
  return {
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
    onReady,
    onDirty: markDirty,
  };
}
