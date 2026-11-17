// Purpose: Render only terminal status and numeric measurements in completion rows.
// Pattern: Dedicated metadata presentation boundary.
// Usage: Live and virtualized history render terminal status and measurements.
// Related: webui/src/pages/workspace/ui/chat/format-statistics.ts, api/meca/v1/state.proto.
import { ChatSystemMessage } from '@astryxdesign/core/Chat';
import { HStack } from '@astryxdesign/core/Layout';
import { StatusDot } from '@astryxdesign/core/StatusDot';
import { Text } from '@astryxdesign/core/Text';
import { Tooltip } from '@astryxdesign/core/Tooltip';
import { memo } from 'react';
import { completionMeasurements } from '@/pages/workspace/ui/chat/format-statistics';
import type { RunCompletionJson } from '@/shared/api';
import { useLocale } from '@/shared/i18n';

export const CompletionRow = memo(function CompletionRow({
  completion,
}: {
  completion: RunCompletionJson;
}) {
  const t = useLocale();
  const completed = completion.status === 'COMPLETED',
    failed = completion.status === 'FAILED';
  const label = failed ? t('failed') : t('interrupted');
  return (
    <ChatSystemMessage className="completion-row min-w-0" aria-label={t('requestMeasurements')}>
      <HStack gap={2} wrap="wrap">
        {!completed && <StatusDot variant={failed ? 'error' : 'warning'} label={label} />}
        <Tooltip content={t('requestMeasurements')}>
          <HStack gap={3} align="center" wrap="wrap" className="completion-summary min-w-0">
            {[
              ...(!completed ? [label] : []),
              ...(completion.statistics ? completionMeasurements(completion.statistics, t) : []),
            ].map((value, index) => (
              <Text
                key={index}
                type="supporting"
                size="base"
                hasTabularNumbers
                color="secondary"
                className="min-w-0 whitespace-normal [overflow-wrap:anywhere]"
              >
                {value}
              </Text>
            ))}
          </HStack>
        </Tooltip>
      </HStack>
    </ChatSystemMessage>
  );
});
