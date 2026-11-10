// Purpose: Edit browser follow-up requests without changing their captured files and references.
// Pattern: Local queue editing surface.
// Usage: Composer displays this queue while the process-owned task continues.
// Related: webui/src/pages/workspace/model/execution.ts, webui/src/pages/workspace/model/store.ts.

import { Button } from '@astryxdesign/core/Button';
import { Dialog, DialogHeader } from '@astryxdesign/core/Dialog';
import { Icon } from '@astryxdesign/core/Icon';
import { HStack, Layout, LayoutContent, VStack } from '@astryxdesign/core/Layout';
import { List, ListItem } from '@astryxdesign/core/List';
import { Text } from '@astryxdesign/core/Text';
import { TextArea } from '@astryxdesign/core/TextArea';
import { type Range, useVirtualizer } from '@tanstack/react-virtual';
import { memo, useCallback, useEffect, useRef, useState } from 'react';
import { queueEditing, resumeQueue } from '@/pages/workspace/model/followups';
import {
  type Queued,
  type Target,
  targetKey,
  useWorkspaceStore,
} from '@/pages/workspace/model/store';
import { focusedRange } from '@/pages/workspace/ui/focused-range';
import { useLocale } from '@/shared/i18n';
export const Queue = memo(function Queue({ target }: { target: Target }) {
  const t = useLocale(),
    entries = useWorkspaceStore((s) => s.queues[targetKey(target)]);
  const [editing, setEditing] = useState<Queued | null>(null),
    [text, setText] = useState('');
  const paused = useWorkspaceStore((s) => s.pausedQueues[targetKey(target)]);
  const scroller = useRef<HTMLDivElement>(null);
  const [focused, setFocused] = useState(-1);
  const editingIndex = entries?.findIndex((value) => value.id === editing?.id) ?? -1;
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
    queueEditing(target, !!editing);
    return () => queueEditing(target, false);
  }, [target, editing]);
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
                  <HStack gap={2}>
                    <Text maxLines={2}>
                      {value.text.slice(0, 256)}
                      {value.text.length > 256 ? '…' : ''}
                    </Text>
                    <Button
                      type="button"
                      className="icon-button"
                      onClick={() => {
                        queueEditing(target, true);
                        setEditing(value);
                        setText(value.text);
                      }}
                      label={t('edit')}
                      variant="ghost"
                    >
                      <Icon icon={'meca:edit'} size="sm" />
                    </Button>
                    <Button
                      type="button"
                      className="icon-button"
                      onClick={() => useWorkspaceStore.getState().removeQueue(target, value.id)}
                      label={t('remove')}
                      variant="ghost"
                    >
                      <Icon icon={'close'} size="sm" />
                    </Button>
                  </HStack>
                }
              />
            );
          })}
        </List>
      </VStack>
      {editing && (
        <Dialog
          isOpen={true}
          onOpenChange={(open) => {
            if (!open) setEditing(null);
          }}
          purpose="form"
          width={760}
          className="queue-modal"
        >
          <Layout
            header={
              <DialogHeader
                title={t('queueTitle')}
                onOpenChange={(open) => {
                  if (!open) setEditing(null);
                }}
              />
            }
            content={
              <LayoutContent padding={4}>
                <VStack gap={4}>
                  <TextArea label={t('input')} value={text} onChange={setText} />
                  <HStack className="form-actions" gap={2}>
                    <Button label={t('cancel')} onClick={() => setEditing(null)} />
                    <Button
                      label={t('save')}
                      variant="primary"
                      onClick={() => {
                        if (useWorkspaceStore.getState().updateQueue(target, editing.id, text))
                          setEditing(null);
                      }}
                    />
                  </HStack>
                </VStack>
              </LayoutContent>
            }
          />
        </Dialog>
      )}
    </VStack>
  );
});
