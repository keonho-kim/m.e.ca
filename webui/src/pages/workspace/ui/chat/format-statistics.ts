// Purpose: Format token counts, elapsed time and throughput in both chat summaries.
// Pattern: Pure presentation module.
// Usage: The composer status line and stored completion rows share these values.
// Related: webui/src/pages/workspace/ui/chat/completion-row.tsx, webui/src/pages/workspace/ui/composer/execution-status.tsx.
import type { StatisticsJson } from '@/shared/api';
import type { useLocale } from '@/shared/i18n';

type Translate = ReturnType<typeof useLocale>;
const counts = new Intl.NumberFormat('en-US');
const rates = new Intl.NumberFormat('en-US', {
  minimumFractionDigits: 1,
  maximumFractionDigits: 1,
});

function displayCount(value: unknown) {
  const parsed = count(value);
  return parsed === undefined ? '—' : counts.format(BigInt(parsed));
}

export function duration(millis: number) {
  const seconds = Number.isFinite(millis) ? Math.max(0, Math.floor(millis / 1000)) : 0;
  return [Math.floor(seconds / 3600), Math.floor(seconds / 60) % 60, seconds % 60]
    .map((value) => String(value).padStart(2, '0'))
    .join(':');
}

export function tokenSummary(usage: StatisticsJson['usage'], t: Translate, showCost = true) {
  const parts = [
    `${t('inputTokens')} ${usage ? displayCount(usage.input) : '—'}`,
    `${t('outputTokens')} ${usage ? displayCount(usage.output) : '—'}`,
  ];
  if (showCost && typeof usage?.cost === 'number' && Number.isFinite(usage.cost) && usage.cost >= 0)
    parts.push(`${t('cost')} $${usage.cost.toFixed(4)}`);
  return parts;
}

export function throughput(stats: StatisticsJson | undefined | null) {
  if (!stats?.usage) return '';
  const millis = Number(count(stats.generationMillis)),
    output = Number(count(stats.usage.output));
  return Number.isSafeInteger(millis) && millis > 0 && Number.isSafeInteger(output) && output >= 0
    ? `${rates.format((1000 * output) / millis)} tok/s`
    : '';
}

function count(value: unknown) {
  if (value === undefined) return '0';
  if (
    typeof value !== 'string' ||
    !/^(0|[1-9]\d{0,18})$/.test(value) ||
    BigInt(value) > 9223372036854775807n
  )
    return undefined;
  return value;
}

export function completionMeasurements(stats: StatisticsJson, t: Translate) {
  return [
    `${t('workedFor')} ${duration(Number(stats.elapsedMillis ?? 0))}`,
    ...tokenSummary(stats.usage, t),
    throughput(stats),
  ].filter(Boolean);
}
