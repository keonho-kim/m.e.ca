// Purpose: Translate user-visible lifecycle notices without exposing internal protocol identifiers.
// Pattern: Pure presentation projection.
// Usage: NoticeRow translates the validated notice body with the active locale.
// Related: webui/src/pages/workspace/ui/chat/notice-row.tsx.
import type { NoticeJson } from '@/shared/api';
import type { useLocale } from '@/shared/i18n';

export function eventNotice(notice: NoticeJson | undefined, t: ReturnType<typeof useLocale>) {
  if (!notice) return undefined;
  if (notice.code === 'SKILL_READ')
    return notice.skill?.name ? t('skillLoaded') + ' · ' + notice.skill.name : undefined;
  const labels = {
    RETRYING: 'retrying',
    COMPACTION_STARTED: 'compacting',
    COMPACTION_FINISHED: 'contextCompacted',
  } as const;
  return notice.code && Object.hasOwn(labels, notice.code)
    ? t(labels[notice.code as keyof typeof labels])
    : undefined;
}
