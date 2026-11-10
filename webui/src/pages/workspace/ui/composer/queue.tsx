// Purpose: Present virtualized follow-ups and bind dispatch prevention to the selected editor.
// Pattern: Queue presentation controller with an independently mounted form.
// Usage: Composer displays pending requests while the process-owned task continues.
// Related: webui/src/pages/workspace/ui/composer/queue-editor.tsx, webui/src/pages/workspace/model/followups.ts.

import { Button } from '@astryxdesign/core/Button';
import { Icon } from '@astryxdesign/core/Icon';
import { IconButton } from '@astryxdesign/core/IconButton';
import { HStack, VStack } from '@astryxdesign/core/Layout';
import { List, ListItem } from '@astryxdesign/core/List';
import { Text } from '@astryxdesign/core/Text';
import { type Range, useVirtualizer } from '@tanstack/react-virtual';
import { memo, useCallback, useEffect, useRef, useState } from 'react';
import { queueEditing, resumeQueue } from '@/pages/workspace/model/followups';
import { type Target, targetKey, useWorkspaceStore } from '@/pages/workspace/model/store';
import { QueueEditor } from '@/pages/workspace/ui/composer/queue-editor';
import { focusedRange } from '@/pages/workspace/ui/focused-range';
import { useLocale } from '@/shared/i18n';
export const Queue = memo(function Queue({ target }: { target: Target }) {
  const t = useLocale(),
    entries = useWorkspaceStore((s) => s.queues[targetKey(target)]);
  const [editing, setEditing] = useState<string | null>(null);
  const entry = entries?.find((value) => value.id === editing);
  const editedID = entry?.id;
  const paused = useWorkspaceStore((s) => s.pausedQueues[targetKey(target)]);
  const scroller = useRef<HTMLDivElement>(null);
  const [focused, setFocused] = useState(-1);
  const editingIndex = entries?.findIndex((value) => value.id === editing) ?? -1;
  const rangeExtractor = useCallback(
    (range: Range) => focusedRange(range, focused, editingIndex),
    [focused, editingIndex]
  );
  const getItemKey = useCallback((index: number) => entries?.[index]?.id ?? index, [entries]);
  const virtual = useVirtualizer({
    count: entries?.length ?? 0,
    getScrollElement: () => scroller.current,
    estimateSize: () => 64,
    overscan: 2,
    getItemKey,
    rangeExtractor,
  });
  useEffect(() => {
    if (editing && !entry) setEditing(null);
  }, [editing, entry]);
  useEffect(() => {
    queueEditing(target, !!editedID);
    return () => queueEditing(target, false);
  }, [target, editedID]);
  if (!entries?.length) return null;
  return (
    <VStack className="queued-inputs" role="region" aria-label={t('queueTitle')} gap={4}>
      <HStack className="queue-heading" gap={2}>
        {paused && (
          <Button label={t('queueResume')} size="sm" onClick={() => resumeQueue(target)} />
        )}
        <Text>
          {t('queueTitle')} ({entries.length})
        </Text>
        <Text type="supporting" color="secondary">
          {t('browserQueue')}
        </Text>
      </HStack>
      <VStack
        ref={scroller}
        style={{ height: `min(24vh, ${virtual.getTotalSize()}px)`, overflowY: 'auto' }}
      >
        <List
          style={{ height: virtual.getTotalSize(), position: 'relative' }}
          onBlurCapture={(event) => {
            if (!event.currentTarget.contains(event.relatedTarget)) setFocused(-1);
          }}
        >
          {virtual.getVirtualItems().map((row) => {
            const value = entries[row.index];
            if (!value) return null;
            return (
              <ListItem
                key={value.id}
                ref={virtual.measureElement}
                data-index={row.index}
                onFocusCapture={() => setFocused(row.index)}
                style={{
                  position: 'absolute',
                  top: 0,
                  left: 0,
                  width: '100%',
                  transform: `translateY(${row.start}px)`,
                }}
                label={
                  <Text maxLines={2} hasTruncateTooltip>
                    {value.text.slice(0, 256) || `${t('attachedImages')} (${value.files.length})`}
                    {value.text.length > 256 ? '…' : ''}
                  </Text>
                }
                endContent={
                  <HStack gap={1}>
                    <IconButton
                      onClick={() => {
                        queueEditing(target, true);
                        setEditing(value.id);
                      }}
                      label={t('edit')}
                      variant="ghost"
                      icon={<Icon icon={'meca:edit'} size="sm" />}
                    />
                    <IconButton
                      onClick={() => useWorkspaceStore.getState().removeQueue(target, value.id)}
                      label={t('remove')}
                      variant="ghost"
                      icon={<Icon icon={'close'} size="sm" />}
                    />
                  </HStack>
                }
              />
            );
          })}
        </List>
      </VStack>
      {entry && (
        <QueueEditor key={entry.id} target={target} entry={entry} close={() => setEditing(null)} />
      )}
    </VStack>
  );
});
