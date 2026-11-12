// Purpose: Show fixed lifecycle notices separately from run measurements and authored messages.
// Pattern: Stateless Astryx notice presentation.
// Usage: HistoryRow passes only a validated notice body.
// Related: webui/src/pages/workspace/model/event-notice.ts, webui/src/pages/workspace/ui/chat/completion-row.tsx.
import { HStack } from '@astryxdesign/core/Layout';
import { Text } from '@astryxdesign/core/Text';
import { eventNotice } from '@/pages/workspace/model/event-notice';
import type { NoticeJson } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
export function NoticeRow({ notice }: { notice: NoticeJson }) {
  const t = useLocale();
  const text = eventNotice(notice, t);
  return text ? (
    <HStack gap={2} className="chat-notice min-w-0">
      <Text type="supporting" color="secondary">
        {text}
      </Text>
    </HStack>
  ) : null;
}
