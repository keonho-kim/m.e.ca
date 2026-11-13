// Purpose: Freeze covered history rendering while preserving focused virtual rows and measured gaps.
// Pattern: Memoized presentation window over the chat-owned virtualizer.
// Usage: Chat continues history acknowledgement and control processing while this view is covered.
// Related: webui/src/pages/workspace/ui/chat/chat.tsx, webui/src/pages/workspace/model/presentation.ts.
import { VStack } from '@astryxdesign/core/Layout';
import { Skeleton } from '@astryxdesign/core/Skeleton';
import type { VirtualItem } from '@tanstack/react-virtual';
import { Fragment, lazy, memo, Suspense } from 'react';
import type { HistoryJump, HistoryRowValue } from '@/pages/workspace/model/history-rows';
import type { InspectionActions } from '@/pages/workspace/model/inspection';
import type { Target } from '@/pages/workspace/model/store';
import { CompletionRow } from '@/pages/workspace/ui/chat/completion-row';

const HistoryRow = lazy(() =>
  import('@/pages/workspace/ui/chat/history-row').then((module) => ({ default: module.HistoryRow }))
);
type Props = {
  rows: HistoryRowValue[];
  window: VirtualItem[];
  total: number;
  measure: (element: Element | null) => void;
  target: Target;
  inspect: InspectionActions;
  match?: HistoryJump;
  setFocused: (index: number) => void;
  blocked: boolean;
};
export const HistoryWindow = memo(
  function HistoryWindow({
    rows,
    window,
    total,
    measure,
    target,
    inspect,
    match,
    setFocused,
  }: Props) {
    return (
      <VStack
        width="100%"
        gap={0}
        className="history-rows"
        onBlurCapture={(event) => {
          if (!event.currentTarget.contains(event.relatedTarget)) setFocused(-1);
        }}
      >
        {window.map((row, index) => {
          const value = rows[row.index];
          if (!value) return null;
          const gap = Math.max(0, row.start - (window[index - 1]?.end ?? 0));
          return (
            <Fragment key={row.key}>
              {gap > 0 && <VStack height={gap} className="shrink-0" aria-hidden gap={0} />}
              <VStack
                onFocusCapture={() => setFocused(row.index)}
                className={`history-row ${'item' in value && value.item.role === 'user' ? 'user-message' : 'assistant-message'}`}
                paddingBlockEnd={8}
                data-index={row.index}
                ref={measure}
              >
                {'completion' in value ? (
                  <CompletionRow
                    completion={value.completion}
                    sequence={value.sequence}
                    onDetails={inspect.history}
                  />
                ) : (
                  <Suspense fallback={<Skeleton height={80} />}>
                    <HistoryRow
                      item={value.item}
                      thinking={value.thinking}
                      tools={value.tools}
                      target={target}
                      inspect={inspect}
                      match={match}
                    />
                  </Suspense>
                )}
              </VStack>
            </Fragment>
          );
        })}
        <VStack
          height={Math.max(0, total - (window.at(-1)?.end ?? 0))}
          className="shrink-0"
          aria-hidden
          gap={0}
        />
      </VStack>
    );
  },
  (previous, next) => next.blocked && previous.target === next.target
);
