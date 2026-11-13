// Purpose: Capture native plain-text edit ranges for exact constant-work typing budgets.
// Pattern: Event-scoped browser adapter with a full-check fallback for composition and tokens.
// Usage: Composer attaches the callback ref to Astryx ChatComposerInput and consumes the next edit.
// Related: webui/src/pages/workspace/model/composer-slice.ts, webui/src/shared/lib/utf8/index.ts.
import { useCallback, useRef } from 'react';
import { useWorkspaceStore } from '@/pages/workspace/model/store';
import { type InputEdit, type Target, targetKey } from '@/pages/workspace/model/types';

export function useInputEdit(target: Target) {
  const pending = useRef<InputEdit | undefined>(undefined);
  const attach = useCallback(
    (root: HTMLDivElement | null) => {
      if (!root) return;
      const capture = (event: Event) => {
        pending.current = undefined;
        if (!(event instanceof InputEvent) || event.isComposing || event.defaultPrevented) return;
        const editable = event.target;
        if (
          !(editable instanceof HTMLElement) ||
          !editable.isContentEditable ||
          editable.childNodes.length !== 1
        )
          return;
        const node = editable.firstChild;
        const range = event.getTargetRanges()[0];
        const value = useWorkspaceStore.getState().inputs[targetKey(target)];
        if (
          !node ||
          node.nodeType !== Node.TEXT_NODE ||
          !range ||
          !value ||
          range.startContainer !== node ||
          range.endContainer !== node ||
          node.textContent?.length !== value.text.length
        )
          return;
        const insert = event.inputType === 'insertText' && event.data !== null;
        const remove =
          event.inputType === 'deleteContentBackward' || event.inputType === 'deleteContentForward';
        if (!insert && !remove) return;
        pending.current = {
          revision: value.revision,
          start: range.startOffset,
          end: range.endOffset,
          inserted: insert ? event.data : '',
        };
        queueMicrotask(() => {
          pending.current = undefined;
        });
      };
      root.addEventListener('beforeinput', capture, true);
      return () => {
        root.removeEventListener('beforeinput', capture, true);
        pending.current = undefined;
      };
    },
    [target]
  );
  const take = () => {
    const value = pending.current;
    pending.current = undefined;
    return value;
  };
  return { attach, take };
}
