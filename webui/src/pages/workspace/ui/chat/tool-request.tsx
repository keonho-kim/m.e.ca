// Purpose: Read tool arguments inside a collapsed, width-aligned request disclosure.
// Pattern: Astryx Collapsible with an owned bounded scroll viewport and lazy detail reads.
// Usage: An expanded ToolEntry mounts this; closing either disclosure releases the request body.
// Related: webui/src/pages/workspace/ui/chat/tool-call.tsx, webui/src/pages/workspace/model/detail-pages.ts.
import { Collapsible } from '@astryxdesign/core/Collapsible';
import { Icon } from '@astryxdesign/core/Icon';
import { HStack } from '@astryxdesign/core/Layout';
import { ScrollableArea } from '@astryxdesign/core/ScrollableArea';
import { Text } from '@astryxdesign/core/Text';
import { useState } from 'react';
import type { Target } from '@/pages/workspace/model/store';
import { toolArguments } from '@/pages/workspace/model/tool-content';
import { DetailContent } from '@/pages/workspace/ui/chat/detail-content';
import { ToolValue } from '@/pages/workspace/ui/chat/tool-value';
import type { EventJson } from '@/shared/api';
import { useLocale } from '@/shared/i18n';

type Props = { target: Target; tool: NonNullable<EventJson['tool']> };
export function ToolRequest({ target, tool }: Props) {
  const t = useLocale(),
    [open, setOpen] = useState(false);
  return (
    <Collapsible
      className="tool-request min-w-0 w-full"
      data-tone="request"
      isOpen={open}
      onOpenChange={setOpen}
      chevronPosition="start"
      trigger={
        <HStack gap={2} align="center" className="min-w-0">
          <Icon icon="meca:request" size="xsm" color="secondary" />
          <Text type="supporting" color="secondary">
            {t('toolRequest')}
          </Text>
        </HStack>
      }
    >
      {open && (
        <ScrollableArea
          label={`${t('toolRequest')} ${tool.name ?? ''}`}
          width="100%"
          maxWidth="100%"
          className="tool-request-content min-w-0"
        >
          <RequestBody target={target} tool={tool} />
        </ScrollableArea>
      )}
    </Collapsible>
  );
}

function RequestBody({ target, tool }: Props) {
  const t = useLocale();
  if (tool.reference)
    return (
      <DetailContent
        key={JSON.stringify(tool.reference)}
        target={target}
        source={{ tool: tool.reference, view: 'TOOL_REQUEST' }}
        request
      />
    );
  const args = toolArguments(tool.argumentsJson);
  return args === undefined ? (
    <Text color="secondary">{t('requestPending')}</Text>
  ) : (
    <ToolValue text={JSON.stringify(args)} request />
  );
}
