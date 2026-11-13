// Purpose: Present parsed tool fields with readable labels and bounded code rather than transport JSON.
// Pattern: Astryx content composition.
// Usage: Inline tool results and nested requests share the same value hierarchy.
// Related: webui/src/pages/workspace/model/tool-content.ts, webui/src/shared/ui/code-preview/index.tsx.

import { Collapsible } from '@astryxdesign/core/Collapsible';
import { HStack, VStack } from '@astryxdesign/core/Layout';
import { Text } from '@astryxdesign/core/Text';
import { useState } from 'react';
import {
  parseToolValue,
  toolFieldName,
  toolFields,
  valueText,
} from '@/pages/workspace/model/tool-content';
import { useLocale } from '@/shared/i18n';
import { CodePreview } from '@/shared/ui/code-preview';

export function ToolValue({ text, request = false }: { text: string; request?: boolean }) {
  const t = useLocale();
  const fieldName = (key: string) => toolFieldName(key, t);
  const value = parseToolValue(text);
  if (!value || typeof value !== 'object' || Array.isArray(value))
    return (
      <CodePreview
        code={valueText(value)}
        language={typeof value === 'string' ? 'plaintext' : 'json'}
      />
    );
  const { content: fields, metadata } = toolFields(value as Record<string, unknown>, request);
  if (fields.length > 32 || (request && fields.length === 0))
    return <CodePreview code={valueText(value)} language="json" />;
  return (
    <VStack gap={5} className="tool-value min-w-0 w-full">
      {fields.map(([key, data]) => {
        const body = valueText(data),
          multiline =
            request || typeof data === 'object' || body.includes('\n') || body.length > 120;
        return multiline ? (
          <VStack key={key} gap={2} className="min-w-0">
            <Text type="supporting" color="secondary">
              {fieldName(key)}
            </Text>
            <CodePreview
              code={body}
              language={
                key.toLowerCase() === 'command'
                  ? 'bash'
                  : typeof data === 'string'
                    ? 'plaintext'
                    : 'json'
              }
            />
          </VStack>
        ) : (
          <HStack key={key} gap={4} align="start" className="min-w-0" wrap="wrap">
            <Text type="supporting" color="secondary">
              {fieldName(key)}
            </Text>
            <Text type="body" className="min-w-0 break-words whitespace-pre-wrap">
              {body}
            </Text>
          </HStack>
        );
      })}
      {!!metadata.length && <ExecutionMetadata fields={metadata} fieldName={fieldName} />}
    </VStack>
  );
}
function ExecutionMetadata({
  fields,
  fieldName,
}: {
  fields: [string, unknown][];
  fieldName: (key: string) => string;
}) {
  const t = useLocale(),
    [open, setOpen] = useState(false);
  return (
    <Collapsible
      isOpen={open}
      onOpenChange={setOpen}
      trigger={
        <Text type="supporting" color="secondary">
          {t('executionInfo')}
        </Text>
      }
    >
      {open &&
        (fields.length > 32 ? (
          <CodePreview code={valueText(Object.fromEntries(fields))} language="json" />
        ) : (
          <VStack gap={2} className="min-w-0">
            {fields.map(([key, data]) => (
              <HStack key={key} gap={3} wrap="wrap" className="min-w-0">
                <Text type="supporting" color="secondary">
                  {fieldName(key)}
                </Text>
                <Text type="supporting" className="break-words min-w-0">
                  {valueText(data)}
                </Text>
              </HStack>
            ))}
          </VStack>
        ))}
    </Collapsible>
  );
}
