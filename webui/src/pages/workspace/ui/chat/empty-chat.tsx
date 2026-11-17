// Purpose: Present the shared landing content for every empty conversation.
// Pattern: Stateless Astryx composition.
// Usage: WorkspaceShell and Chat share this content before the first message.
// Related: webui/src/pages/workspace/ui/chat/layout.ts, webui/src/pages/workspace/ui/chat/chat.tsx.
import { Heading } from '@astryxdesign/core/Heading';
import { VStack } from '@astryxdesign/core/Layout';
import { Text } from '@astryxdesign/core/Text';
import { useLocale } from '@/shared/i18n';

export function EmptyChat({ hasWorkspace = true }: { hasWorkspace?: boolean }) {
  const t = useLocale();
  return (
    <VStack gap={3} paddingBlock={6}>
      <Heading level={2} type="display-2">
        {t('emptyTitle')}
      </Heading>
      <Text color="secondary">{t(hasWorkspace ? 'emptyHelp' : 'selectWorkspace')}</Text>
    </VStack>
  );
}
