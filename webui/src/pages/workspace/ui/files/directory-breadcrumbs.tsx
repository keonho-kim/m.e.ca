// Purpose: Keep the directory trail within its panel and collapse older ancestors first.
// Pattern: Astryx Breadcrumbs with the official overflow measurement hook.
// Usage: DirectoryColumn supplies navigation; only the last three named ancestors are measured.
// Related: webui/src/pages/workspace/model/directory-trail.ts, webui/src/pages/workspace/ui/files/directory-column.tsx.
import { BreadcrumbItem, Breadcrumbs } from '@astryxdesign/core/Breadcrumbs';
import { useOverflow } from '@astryxdesign/core/hooks';
import { HStack } from '@astryxdesign/core/Layout';
import { Text } from '@astryxdesign/core/Text';
import { directoryTrail } from '@/pages/workspace/model/directory-trail';
import { useLocale } from '@/shared/i18n';

export function DirectoryBreadcrumbs({
  path,
  open,
  busy,
}: {
  path: string;
  open: (path: string) => void;
  busy: boolean;
}) {
  const t = useLocale();
  const trail = directoryTrail(path);
  const candidates = trail.slice(-3);
  const overflow = useOverflow(candidates.length, {
    minVisibleItems: 1,
    maxVisibleItems: 3,
    collapseFrom: 'start',
  });
  const count = Math.min(candidates.length, overflow.visibleCount);
  const visible = count ? candidates.slice(-count) : [];
  const hidden = trail.slice(0, trail.length - count);
  if (!trail.length) return null;
  return (
    <HStack width="80%" className="relative min-w-0" gap={0}>
      <Breadcrumbs
        ref={overflow.containerRef}
        label={t('fileWorkspace')}
        variant="default"
        className="min-w-0 w-full directory-breadcrumbs"
      >
        {!!hidden.length && (
          <BreadcrumbItem
            menu={hidden.map((entry) => ({
              label: entry.label,
              isDisabled: busy,
              onClick: () => open(entry.path),
            }))}
          >
            …
          </BreadcrumbItem>
        )}
        {visible.map((entry, index) => (
          <BreadcrumbItem
            key={entry.path}
            isCurrent={index === visible.length - 1}
            onClick={busy ? undefined : () => open(entry.path)}
            className="min-w-0"
          >
            <Text type="body" maxLines={1} hasTruncateTooltip className="min-w-0">
              {entry.label}
            </Text>
          </BreadcrumbItem>
        ))}
      </Breadcrumbs>
      <HStack
        ref={overflow.measureRef}
        gap={0}
        className="absolute invisible pointer-events-none h-0 overflow-hidden w-max"
        aria-hidden
        inert
      >
        {candidates.map((entry) => (
          <HStack key={entry.path} gap={2} className="shrink-0">
            <Text type="body">{entry.label}</Text>
            <Text type="body">/</Text>
          </HStack>
        ))}
        <HStack data-overflow-indicator gap={2}>
          <Text type="body">…</Text>
          <Text type="body">/</Text>
        </HStack>
      </HStack>
    </HStack>
  );
}
