// Purpose: Coordinate virtual history measurement and reader-position restoration with Astryx follow ownership.
// Pattern: Conversation-owned viewport hook.
// Usage: Chat supplies saved rows; the hook releases its position entry through bounded retention.
// Related: webui/src/pages/workspace/ui/chat/chat.tsx, webui/patches/@astryxdesign%2Fcore@0.7.17.patch.
import { useChatLayoutContext } from '@astryxdesign/core/Chat';
import { type Range, useVirtualizer } from '@tanstack/react-virtual';
import { useCallback, useLayoutEffect, useState } from 'react';
import { toolBatchIdentity, toolIdentity } from '@/pages/workspace/model/chat-events';
import type { HistoryRowValue } from '@/pages/workspace/model/history-rows';
import { type Target, targetKey } from '@/pages/workspace/model/store';
import { focusedRange } from '@/pages/workspace/ui/focused-range';

const positions = new Map<string, number>();
export function useHistoryViewport(
  visibleRows: HistoryRowValue[],
  scroller: HTMLDivElement | null,
  target: Target
) {
  const scroll = useChatLayoutContext()?.scroll;
  const [focused, setFocused] = useState(-1);
  const getItemKey = useCallback(
    (index: number) => {
      const row = visibleRows[index];
      return (
        (row &&
          ('item' in row
            ? (row.item.event &&
                (toolBatchIdentity(row.item.event) || toolIdentity(row.item.event))) ||
              row.item.presentation ||
              row.item.sequence
            : row.sequence)) ??
        index
      );
    },
    [visibleRows]
  );
  const rangeExtractor = useCallback((range: Range) => focusedRange(range, focused), [focused]);
  const virtual = useVirtualizer({
    count: visibleRows.length,
    getScrollElement: () => scroller,
    estimateSize: () => 140,
    overscan: 5,
    getItemKey,
    rangeExtractor,
  });
  virtual.shouldAdjustScrollPositionOnItemSizeChange = (item, _delta, instance) => {
    if (scroll?.isFollowing()) return false;
    const offset = instance.scrollOffset ?? 0;
    return !instance.itemSizeCache.has(item.key)
      ? item.start < offset
      : item.end <= offset && instance.scrollDirection !== 'backward';
  };
  useLayoutEffect(() => {
    const element = scroller;
    if (!element) return;
    const saved = positions.get(targetKey(target));
    if (saved !== undefined) {
      scroll?.unlock();
      element.scrollTop = saved;
      if (element.scrollHeight - element.clientHeight - saved < 1)
        scroll?.scrollToBottom({ behavior: 'instant' });
    }
    return () => {
      positions.set(targetKey(target), element.scrollTop);
      if (positions.size > 12) positions.delete(positions.keys().next().value!);
    };
  }, [target, scroller, scroll]);
  return { virtual, scroll, setFocused };
}
