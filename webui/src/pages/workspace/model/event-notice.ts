// Purpose: Translate user-visible lifecycle notices without exposing internal protocol identifiers.
// Pattern: Pure presentation projection.
// Usage: Stored history and live activity share the same narrow notice vocabulary.
// Related: webui/src/pages/workspace/ui/chat/history-row.tsx, webui/src/pages/workspace/ui/chat/live-activity.tsx.
import type { EventJson } from '@/shared/api';
import type { useLocale } from '@/shared/i18n';

export function eventNotice(event: EventJson, t: ReturnType<typeof useLocale>) {
  if (event.skill?.name) return `${t('skillLoaded')} · ${event.skill.name}`;
  const labels = {
    retry: 'retrying',
    compaction_started: 'compacting',
    compaction_finished: 'contextCompacted',
    team_work: 'teamUpdate',
  } as const;
  if (!event.kind || !Object.hasOwn(labels, event.kind)) return undefined;
  const label = labels[event.kind as keyof typeof labels];
  return [t(label), event.text?.slice(0, 1024)].filter(Boolean).join(' · ');
}
