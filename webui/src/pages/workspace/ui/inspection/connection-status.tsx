// Purpose: Show one accurate SSE state and offer stream display controls and execution details.
// Pattern: Independent Astryx status subscriber.
// Usage: WorkspaceShell owns this single connection status location.
// Related: webui/src/pages/workspace/api/events.ts, webui/src/pages/workspace/ui/inspection/metrics.tsx.
import { Icon } from '@astryxdesign/core/Icon';
import { IconButton } from '@astryxdesign/core/IconButton';
import { HStack } from '@astryxdesign/core/Layout';
import { Popover } from '@astryxdesign/core/Popover';
import { StatusDot } from '@astryxdesign/core/StatusDot';
import { Text } from '@astryxdesign/core/Text';
import { useState } from 'react';
import { type Target, useWorkspaceStore } from '@/pages/workspace/model/store';
import { Metrics } from '@/pages/workspace/ui/inspection/metrics';
import { useLocale } from '@/shared/i18n';
export function ConnectionStatus({ target }: { target: Target }) {
  const [metricsOpen, setMetricsOpen] = useState(false);
  const t = useLocale(),
    status = useWorkspaceStore((s) => s.streamStatus),
    paused = useWorkspaceStore((s) => s.paused);
  const label = t(
    paused
      ? 'paused'
      : status === 'connected'
        ? 'eventsConnected'
        : status === 'connecting'
          ? 'eventsConnecting'
          : 'eventsReconnecting'
  );
  return (
    <HStack gap={2} paddingInline={4} paddingBlock={1} justify="end">
      <Popover
        label={t('executionInfo')}
        isOpen={metricsOpen}
        onOpenChange={setMetricsOpen}
        placement="above"
        width={360}
        content={metricsOpen ? <Metrics target={target} /> : null}
      >
        <IconButton
          label={t('executionInfo')}
          variant="ghost"
          size="sm"
          icon={<Icon icon={'info'} size="xsm" />}
        />
      </Popover>
      {!!target.workspace && (
        <>
          <StatusDot
            variant={
              paused
                ? 'warning'
                : status === 'connected'
                  ? 'success'
                  : status === 'connecting'
                    ? 'neutral'
                    : 'warning'
            }
            label={label}
            tooltip={label}
            aria-hidden={paused || status !== 'connected'}
          />
          {(paused || status !== 'connected') && (
            <Text type="supporting" color="secondary" role="status">
              {label}
            </Text>
          )}
        </>
      )}
      <IconButton
        label={paused ? t('resume') : t('pause')}
        variant="ghost"
        size="sm"
        isDisabled={!target.workspace}
        icon={<Icon icon={paused ? 'meca:play' : 'meca:pause'} size="xsm" />}
        onClick={() => useWorkspaceStore.getState().set({ paused: !paused })}
      />
    </HStack>
  );
}
