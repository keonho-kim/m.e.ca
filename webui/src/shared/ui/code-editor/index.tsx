// Purpose: Own bounded text input, undo transactions and cancellable syntax presentation.
// Pattern: Controlled editor adapter with explicit captured-save acknowledgement.
// Usage: FileDocument owns persistence and keeps this editor mounted across Preview/Raw changes.
// Related: webui/src/shared/ui/code-editor/history.ts, webui/src/shared/ui/code-editor/syntax.ts, webui/src/shared/ui/code-editor/input.ts.
import { VStack } from '@astryxdesign/core/Layout';
import {
  memo,
  type Ref,
  useCallback,
  useEffect,
  useImperativeHandle,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import SimpleEditor from 'react-simple-code-editor';
import { codeLanguage } from '@/shared/lib/code-language';
import { failure } from '@/shared/lib/errors';
import { type SimpleEditorHandle, trimEditorHistory } from '@/shared/ui/code-editor/history';
import { editorTextAdmission } from '@/shared/ui/code-editor/input';
import { canHighlight, paintSyntax, syntaxClasses } from '@/shared/ui/code-editor/syntax';

export type EditorHandle = {
  text: () => string;
  capture: () => { text: string; saved: () => void };
  focus: () => void;
};
const plainText = (value: string) => (
  <>
    {value}
    {'\n'}
  </>
);

const Editor = memo(function Editor({
  initialText,
  path,
  maxBytes,
  onDirty,
  onReady,
  onError,
  onDraft,
  visible,
  ref,
}: {
  initialText: string;
  path: string;
  maxBytes: number;
  onDirty: (dirty: boolean) => void;
  onReady: () => void;
  onError: (error: unknown) => void;
  onDraft: (value: string) => void;
  visible: boolean;
  ref: Ref<EditorHandle>;
}) {
  const [text, setText] = useState(initialText),
    [composing, setComposing] = useState(false);
  const current = useRef(initialText),
    saved = useRef(initialText),
    dirty = useRef(false);
  const root = useRef<HTMLDivElement>(null),
    library = useRef<SimpleEditorHandle>(null);
  const trimPending = useRef(false),
    rejected = useRef(false);
  const notify = useRef(onDirty),
    ready = useRef(onReady),
    failed = useRef(onError);
  notify.current = onDirty;
  ready.current = onReady;
  failed.current = onError;
  const language = useMemo(() => codeLanguage(path), [path]);
  const plain = useMemo(() => !canHighlight(text), [text]);
  const updateDirty = useCallback(() => {
    const next = current.current !== saved.current;
    if (dirty.current === next) return;
    dirty.current = next;
    notify.current(next);
  }, []);
  const trimHistory = useCallback(() => {
    if (trimPending.current) return;
    trimPending.current = true;
    // Undo and composition update their offsets after the value callback.
    queueMicrotask(() => {
      trimPending.current = false;
      if (library.current) trimEditorHistory(library.current.session.history);
    });
  }, []);
  const fits = useMemo(() => editorTextAdmission(maxBytes), [maxBytes]);
  const accept = useCallback(
    (value: string) => {
      const valid = fits(value);
      if (!valid && !rejected.current) failed.current(failure('editor_limit'));
      rejected.current = !valid;
      return valid;
    },
    [fits]
  );
  useEffect(() => {
    root.current?.querySelector('textarea')?.setAttribute('aria-label', path);
    ready.current();
  }, [path]);
  useLayoutEffect(() => {
    const input = root.current?.querySelector('textarea');
    if (!input) return;
    const beforeInput = (event: InputEvent) => {
      const inserted = event.data ?? event.dataTransfer?.getData('text/plain');
      if (
        !event.cancelable ||
        event.isComposing ||
        event.inputType.includes('Composition') ||
        !inserted ||
        !event.inputType.startsWith('insert') ||
        (inserted.length < 1024 && !inserted.includes('\n'))
      )
        return;
      // Native bulk insertion can rebuild thousands of editable line nodes one at a time.
      event.preventDefault();
      if (inserted.length > maxBytes) accept(inserted);
      else library.current?.replaceSelection(inserted);
    };
    input.addEventListener('beforeinput', beforeInput);
    return () => input.removeEventListener('beforeinput', beforeInput);
  }, [accept, maxBytes]);
  useEffect(() => {
    const element = root.current?.querySelector('pre');
    if (!element || !visible || composing) return;
    const controller = new AbortController();
    let cleanup: (() => void) | undefined;
    const timer = setTimeout(() => {
      void paintSyntax(element, text, language, controller.signal).then(
        (dispose) => {
          if (controller.signal.aborted) dispose?.();
          else cleanup = dispose;
        },
        (error: unknown) => {
          if (!controller.signal.aborted) failed.current(error);
        }
      );
    }, 80);
    return () => {
      clearTimeout(timer);
      controller.abort();
      cleanup?.();
    };
  }, [language, text, visible, composing]);
  useImperativeHandle(
    ref,
    () => ({
      focus: () => root.current?.querySelector('textarea')?.focus(),
      text: () => current.current,
      capture: () => {
        library.current?.closeHistoryGroup();
        const captured = current.current;
        return {
          text: captured,
          saved: () => {
            saved.current = captured;
            updateDirty();
          },
        };
      },
    }),
    [updateDirty]
  );
  return (
    <VStack
      ref={root}
      gap={0}
      height="100%"
      className="min-h-0 min-w-0 overflow-auto"
      onPasteCapture={(event) => {
        const pasted = event.clipboardData.getData('text/plain');
        if (!pasted) return;
        event.preventDefault();
        if (pasted.length > maxBytes) accept(pasted);
        else library.current?.replaceSelection(pasted);
      }}
      onCompositionStartCapture={() => setComposing(true)}
      onCompositionEndCapture={() => {
        setComposing(false);
        trimHistory();
      }}
    >
      <SimpleEditor
        ref={library}
        plain={plain}
        acceptValue={accept}
        maxLength={maxBytes}
        className="text-editor min-h-full font-mono text-sm leading-relaxed"
        style={{ flex: plain ? '1 1 0%' : '0 0 auto', minWidth: 0, width: '100%' }}
        preClassName={syntaxClasses}
        textareaClassName="focus-visible:outline-2 focus-visible:outline-(--focus-outline-color) focus-visible:-outline-offset-2"
        padding="var(--spacing-5)"
        value={text}
        highlight={plainText}
        onValueChange={(value) => {
          current.current = value;
          setText(value);
          if (value !== saved.current) onDraft(value);
          updateDirty();
          trimHistory();
        }}
      />
    </VStack>
  );
});
export default Editor;
