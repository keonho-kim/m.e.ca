// Purpose: Render only terminal status and numeric measurements in completion rows.
// Pattern: Dedicated metadata presentation boundary.
// Usage: Chat supplies measurements and owns the lifetime of requested completion details.
// Related: webui/src/pages/workspace/ui/chat/format-statistics.ts, api/meca/v1/state.proto.
import { Button } from '@astryxdesign/core/Button';
import { ChatSystemMessage } from '@astryxdesign/core/Chat';
import { HStack } from '@astryxdesign/core/Layout';
import { StatusDot } from '@astryxdesign/core/StatusDot';
import { Text } from '@astryxdesign/core/Text';
import { memo } from 'react';
import { completionMeasurements } from '@/pages/workspace/ui/chat/format-statistics';
import type { RunCompletionJson } from '@/shared/api';
import { useLocale } from '@/shared/i18n';

export const CompletionRow = memo(function CompletionRow({
  completion,
  sequence,
  onDetails,
}: {
  completion: RunCompletionJson;
  sequence?: string;
  onDetails: (sequence: string) => void;
}) {
  const t = useLocale();
  const completed = completion.status === 'COMPLETED',
    failed = completion.status === 'FAILED';
  const label = completed ? t('completed') : failed ? t('failed') : t('interrupted');
  return (
    <ChatSystemMessage className="completion-row min-w-0">
      <HStack gap={2} wrap="wrap">
        {!completed && <StatusDot variant={failed ? 'error' : 'warning'} label={label} />}
        <HStack gap={3} align="center" wrap="wrap" className="completion-summary min-w-0">
          {[
            label,
            ...(completion.statistics ? completionMeasurements(completion.statistics, t) : []),
          ].map((value, index) => (
            <Text key={index} type="supporting" color="secondary" className="whitespace-nowrap">
              {value}
            </Text>
          ))}
        </HStack>
        {!completed && sequence && (
          <Button label={t('details')} size="sm" onClick={() => onDetails(sequence)} />
        )}
      </HStack>
    </ChatSystemMessage>
  );
});
