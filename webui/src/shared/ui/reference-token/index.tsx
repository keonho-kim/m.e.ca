// Purpose: Give files, folders and skills the same icon-bearing Astryx chat-token presentation.
// Pattern: Shared token composition.
// Usage: Composer references, inline skill triggers and user message text share this definition.
// Related: webui/src/pages/workspace/ui/composer/composer.tsx, webui/src/pages/workspace/ui/chat/history-row.tsx.
import {
  type ChatComposerToken,
  ChatComposerTokenElement,
  ChatTokenizedText,
} from '@astryxdesign/core/Chat';
import { Icon } from '@astryxdesign/core/Icon';
import { Text } from '@astryxdesign/core/Text';
import { Tooltip } from '@astryxdesign/core/Tooltip';

export function referenceToken(value: string, label = value): ChatComposerToken {
  return {
    value,
    label: (
      <Text as="span" className="min-w-0 whitespace-normal [overflow-wrap:anywhere]">
        {label}
      </Text>
    ),
    icon: (
      <Icon
        icon={
          value.startsWith('#') ? 'meca:book' : value.endsWith('/') ? 'meca:folder' : 'meca:file'
        }
      />
    ),
    variant: 'neutral',
  };
}
export function ReferenceText({ text }: { text: string }) {
  const values = [...new Set(text.match(/(?:#[\w:.-]+|^![\w-]+)/g) ?? [])].slice(0, 128);
  return (
    <ChatTokenizedText
      className="min-w-0 whitespace-pre-wrap [overflow-wrap:anywhere]"
      tokens={values.map((value) => ({
        value,
        render: () => <ChatComposerTokenElement token={referenceToken(value)} />,
      }))}
    >
      {text}
    </ChatTokenizedText>
  );
}
export function ReferenceToken({ value }: { value: string }) {
  return (
    <Tooltip content={value}>
      <ChatComposerTokenElement
        className="reference-token max-w-full min-w-0"
        token={referenceToken(value)}
      />
    </Tooltip>
  );
}
