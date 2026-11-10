// Purpose: Bind Astryx CodeEditor to revision-bound saves without losing edits made during a save.
// Pattern: Controlled editor adapter with explicit captured-save acknowledgement.
// Usage: FileDocument lazy-loads the official Lab component and owns persistence.
// Related: webui/src/pages/workspace/ui/files/file-document.tsx, webui/src/shared/lib/code-language/index.ts.

import { CodeEditor } from '@astryxdesign/lab';
import { type Ref, useEffect, useImperativeHandle, useRef, useState } from 'react';
import { codeLanguage } from '@/shared/lib/code-language';
import '@astryxdesign/lab/lab.css';
export type EditorHandle = { capture: () => { text: string; saved: () => void } };
export default function Editor({
  initialText,
  path,
  onDirty,
  onReady,
  ref,
}: {
  initialText: string;
  path: string;
  onDirty: (dirty: boolean) => void;
  onReady: () => void;
  ref: Ref<EditorHandle>;
}) {
  const [text, setText] = useState(initialText),
    current = useRef(initialText),
    saved = useRef(initialText);
  const notify = useRef(onDirty),
    ready = useRef(onReady);
  notify.current = onDirty;
  ready.current = onReady;
  useEffect(() => {
    ready.current();
  }, []);
  useImperativeHandle(
    ref,
    () => ({
      capture: () => {
        const captured = current.current;
        return {
          text: captured,
          saved: () => {
            saved.current = captured;
            notify.current(current.current !== captured);
          },
        };
      },
    }),
    []
  );
  return (
    <CodeEditor
      label={path}
      className="text-editor"
      value={text}
      language={codeLanguage(path)}
      hasLineNumbers={text.length <= 65536 && text.split('\n', 2001).length <= 2000}
      onChange={(value) => {
        current.current = value;
        setText(value);
        notify.current(value !== saved.current);
      }}
    />
  );
}
