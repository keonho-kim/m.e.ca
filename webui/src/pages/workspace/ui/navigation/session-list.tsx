// Purpose: Render a bounded visible window of project conversations with native Astryx actions.
// Pattern: Virtualized presentation list.
// Usage: WorkspaceSessions supplies one bounded cursor window and scoped navigation callbacks.
// Related: webui/src/pages/workspace/ui/navigation/workspace-sessions.tsx.
import { Badge } from '@astryxdesign/core/Badge';
import { Button } from '@astryxdesign/core/Button';
import { Icon } from '@astryxdesign/core/Icon';
import { IconButton } from '@astryxdesign/core/IconButton';
import { HStack, VStack } from '@astryxdesign/core/Layout';
import { SideNavItem, useSideNavCollapse } from '@astryxdesign/core/SideNav';
import { StatusDot } from '@astryxdesign/core/StatusDot';
import { Tooltip } from '@astryxdesign/core/Tooltip';
import { type Range, useVirtualizer } from '@tanstack/react-virtual';
import { memo, useCallback, useEffect, useRef, useState } from 'react';
import { type Target, useWorkspaceStore } from '@/pages/workspace/model/store';
import { focusedRange } from '@/pages/workspace/ui/focused-range';
import type { SessionJson } from '@/shared/api';
import { useLocale } from '@/shared/i18n';

export const SessionList = memo(function SessionList({
  items,
  workspace,
  target,
  running,
  navigate,
  remove,
}: {
  items: SessionJson[];
  workspace: string;
  target: Target;
  running: Set<string>;
  navigate: (target: Target) => void;
  remove: (session: SessionJson) => void;
}) {
  const t = useLocale(),
    scroller = useRef<HTMLDivElement>(null),
    selected = useRef('');
  const [focused, setFocused] = useState(-1);
  const getItemKey = useCallback((index: number) => items[index]?.id ?? index, [items]);
  const rangeExtractor = useCallback((range: Range) => focusedRange(range, focused), [focused]);
  const virtual = useVirtualizer({
    count: items.length,
    getScrollElement: () => scroller.current,
    estimateSize: () => 40,
    overscan: 3,
    getItemKey,
    rangeExtractor,
  });
  useEffect(() => {
    if (selected.current === target.session) return;
    const index = items.findIndex((item) => item.id === target.session);
    if (index < 0) return;
    selected.current = target.session;
    virtual.scrollToIndex(index, { align: 'auto' });
  }, [items, target.session, virtual]);
  return (
    <VStack
      ref={scroller}
      role="group"
      aria-label={t('conversations')}
      tabIndex={0}
      style={{ height: `min(45vh, ${virtual.getTotalSize()}px)`, overflowY: 'auto' }}
      onBlurCapture={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget)) setFocused(-1);
      }}
    >
      <VStack gap={0} style={{ height: virtual.getTotalSize(), position: 'relative' }}>
        {virtual.getVirtualItems().map((row) => {
          const session = items[row.index];
          if (!session) return null;
          return (
            <VStack
              key={row.key}
              data-index={row.index}
              ref={virtual.measureElement}
              onFocusCapture={() => setFocused(row.index)}
              style={{
                position: 'absolute',
                top: 0,
                left: 0,
                width: '100%',
                transform: `translateY(${row.start}px)`,
              }}
            >
              <SessionRow
                session={session}
                workspace={workspace}
                target={target}
                running={running.has(session.id ?? '')}
                navigate={navigate}
                remove={remove}
              />
            </VStack>
          );
        })}
      </VStack>
    </VStack>
  );
});

function SessionRow({
  session,
  workspace,
  target,
  running,
  navigate,
  remove,
}: {
  session: SessionJson;
  workspace: string;
  target: Target;
  running: boolean;
  navigate: (target: Target) => void;
  remove: (session: SessionJson) => void;
}) {
  const t = useLocale(),
    label = session.summary || t('untitled');
  const labelRef = useRef<HTMLElement>(null);
  const rail = useSideNavCollapse();
  return (
    <>
      <SideNavItem
        ref={labelRef}
        label={label}
        className="session-link session-row"
        icon={<Icon icon="meca:message" size="xsm" />}
        isSelected={target.workspace === workspace && target.session === session.id}
        onClick={() => navigate({ workspace, session: session.id ?? '' })}
        endContent={
          session.runningTaskId || running ? (
            <StatusDot variant="accent" label={t('running')} isPulsing />
          ) : undefined
        }
        actions={
          <HStack gap={1}>
            {!!session.pendingApprovals && (
              <Button
                label={t('approval')}
                size="sm"
                variant="ghost"
                onClick={() =>
                  useWorkspaceStore
                    .getState()
                    .set({ reviewTarget: { workspace, session: session.id ?? '' } })
                }
                endContent={<Badge label={session.pendingApprovals} />}
              />
            )}
            <IconButton
              label={t('deleteChat')}
              size="sm"
              variant="ghost"
              icon={<Icon icon="meca:delete" size="xsm" />}
              onClick={() => remove(session)}
            />
          </HStack>
        }
      />
      <Tooltip anchorRef={labelRef} content={label} isEnabled={!rail.isCollapsed} />
    </>
  );
}
