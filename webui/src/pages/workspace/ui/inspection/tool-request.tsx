// Purpose: Show tool arguments as a readable modal without mixing in execution output or transcript metadata.
// Pattern: Native Astryx Dialog over reference-only inspection state.
// Usage: The conversation inspector mounts this only after a tool's request icon is selected.
// Related: webui/src/pages/workspace/model/inspection.ts, webui/src/pages/workspace/ui/chat/detail-content.tsx.
import { Dialog, DialogHeader } from '@astryxdesign/core/Dialog';
import { Layout, LayoutContent } from '@astryxdesign/core/Layout';
import { Text } from '@astryxdesign/core/Text';
import type { ToolRequestInspection } from '@/pages/workspace/model/inspection';
import type { Target } from '@/pages/workspace/model/store';
import { DetailContent } from '@/pages/workspace/ui/chat/detail-content';
import { ToolValue } from '@/pages/workspace/ui/chat/tool-value';
import { useLocale } from '@/shared/i18n';
export function ToolRequest({
  value,
  target,
  close,
}: {
  value: ToolRequestInspection;
  target: Target;
  close: () => void;
}) {
  const t = useLocale();
  return (
    <Dialog
      isOpen
      onOpenChange={(open) => {
        if (!open) close();
      }}
      width={880}
      padding={0}
      className="tool-request-modal"
    >
      <Layout
        height="auto"
        padding={6}
        header={
          <DialogHeader
            title={value.name}
            subtitle={t('toolRequest')}
            onOpenChange={(open) => {
              if (!open) close();
            }}
          />
        }
        content={
          <LayoutContent padding={6}>
            {value.reference ? (
              <DetailContent
                target={target}
                source={{ tool: value.reference, view: 'TOOL_REQUEST' }}
                request
              />
            ) : value.args === undefined ? (
              <Text color="secondary">{t('requestPending')}</Text>
            ) : (
              <ToolValue text={JSON.stringify(value.args, null, 2)} request />
            )}
          </LayoutContent>
        }
      />
    </Dialog>
  );
}
