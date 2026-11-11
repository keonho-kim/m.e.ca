// Purpose: Format token counts, elapsed time and throughput in both chat summaries.
// Pattern: Pure presentation module.
// Usage: The composer status line and stored completion rows share these values.
// Related: webui/src/pages/workspace/ui/chat/history-row.tsx, webui/src/pages/workspace/ui/composer/execution-status.tsx.
import type { StatisticsJson } from '@/shared/api';
import type { useLocale } from '@/shared/i18n';

type Translate = ReturnType<typeof useLocale>;

export function duration(millis: number) {
  const seconds = Number.isFinite(millis) ? Math.max(0, Math.floor(millis / 1000)) : 0;
  return [Math.floor(seconds / 3600), Math.floor(seconds / 60) % 60, seconds % 60]
    .map((value) => String(value).padStart(2, '0'))
    .join(':');
}

export function tokenSummary(usage: StatisticsJson['usage'], t: Translate, showCost = true) {
  const parts = [
    `${t('inputTokens')} ${usage ? (usage.input ?? '0') : '—'}`,
    `${t('outputTokens')} ${usage ? (usage.output ?? '0') : '—'}`,
  ];
  if (showCost && typeof usage?.cost === 'number')
    parts.push(`${t('cost')} $${usage.cost.toFixed(4)}`);
  return parts;
}

export function throughput(stats: StatisticsJson | undefined | null) {
  if (!stats?.usage) return '';
  const millis = Number(stats.generationMillis ?? 0);
  const count = Number(stats.usage.output ?? 0);
  return millis > 0 && count >= 0 ? `${((1000 * count) / millis).toFixed(1)} tok/s` : '';
}

export function completionSummary(stats: StatisticsJson, t: Translate) {
  return [
    `${t('workedFor')} ${duration(Number(stats.elapsedMillis ?? 0))}`,
    ...tokenSummary(stats.usage, t),
    throughput(stats),
  ]
    .filter(Boolean)
    .join(' · ');
}
