// Purpose: Keep oversized tool objects readable through independently paged fields.
// Pattern: Lazy semantic disclosure over the native detail projection.
// Usage: DetailContent uses this only when a complete object exceeds its byte page.
// Related: internal/serverapi/detail/fields.go, webui/src/pages/workspace/model/detail-pages.ts.
import { Collapsible } from '@astryxdesign/core/Collapsible';
import { VStack } from '@astryxdesign/core/Layout';
import { Skeleton } from '@astryxdesign/core/Skeleton';
import { Text } from '@astryxdesign/core/Text';
import { type ReactNode, useMemo, useState } from 'react';
import { useDetailPage } from '@/pages/workspace/model/detail-pages';
import type { Target } from '@/pages/workspace/model/store';
import { toolFieldName } from '@/pages/workspace/model/tool-content';
import { DetailContent } from '@/pages/workspace/ui/chat/detail-content';
import type { DetailRequestJson } from '@/shared/api';
import { useLocale } from '@/shared/i18n';

type Props = { target: Target; source: DetailRequestJson };
export function StructuredTool({ target, source, fallback }: Props & { fallback: ReactNode }) {
  const { page } = useDetailPage(target, { ...source, listFields: true });
  const [expanded, setExpanded] = useState<string[] | null>(null);
  const fields = useMemo(() => {
    try {
      const keys: unknown = JSON.parse(page.data?.text ?? '[]');
      return Array.isArray(keys) &&
        keys.length <= 64 &&
        keys.every((key) => typeof key === 'string' && key.length <= 512)
        ? (keys as string[])
        : [];
    } catch {
      return [];
    }
  }, [page.data?.text]);
  if (page.isPending) return <Skeleton height={80} />;
  if (!fields.length) return fallback;
  const opened =
    expanded ??
    fields
      .filter((field) => ['stdout', 'stderr', 'content', 'text', 'output'].includes(field))
      .slice(0, 2);
  return (
    <VStack gap={4} width="100%">
      {fields.map((field) => (
        <Field
          key={field}
          target={target}
          source={source}
          name={field}
          open={opened.includes(field)}
          change={(open) =>
            setExpanded(
              open
                ? [...opened.filter((name) => name !== field), field].slice(-4)
                : opened.filter((name) => name !== field)
            )
          }
        />
      ))}
    </VStack>
  );
}
function Field({
  target,
  source,
  name,
  open,
  change,
}: Props & { name: string; open: boolean; change: (open: boolean) => void }) {
  const t = useLocale();
  return (
    <Collapsible
      isOpen={open}
      onOpenChange={change}
      className="min-w-0 w-full"
      data-tone="request"
      trigger={<Text type="supporting">{toolFieldName(name, t)}</Text>}
    >
      {open && <DetailContent target={target} source={{ ...source, field: name }} />}
    </Collapsible>
  );
}
