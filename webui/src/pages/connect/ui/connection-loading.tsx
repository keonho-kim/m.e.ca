// Purpose: Present cookie restoration and workspace startup in the application's glass material.
// Pattern: Stateless app loading presentation.
// Usage: App borrows the connection page's handoff view for session checks and lazy workspace startup.
// Related: webui/src/app/routes/application.tsx, webui/src/app/styles/theme.ts, webui/src/shared/ui/wordmark/index.tsx.
import { Center } from '@astryxdesign/core/Center';
import { HStack, VStack } from '@astryxdesign/core/Layout';
import { Spinner } from '@astryxdesign/core/Spinner';
import { Text } from '@astryxdesign/core/Text';
import { useLocale } from '@/shared/i18n';
import { Wordmark } from '@/shared/ui/wordmark';

export function ConnectionLoading({ checking }: { checking: boolean }) {
  const t = useLocale();
  return (
    <Center minHeight="100dvh" width="100%" padding={6} role="status" className="startup-screen">
      <VStack width="100%" maxWidth={440} padding={8} gap={6} className="startup-surface">
        <HStack gap={4} align="center" justify="between">
          <Wordmark />
          <Spinner size="xl" shade="subtle" aria-hidden />
        </HStack>
        <Text color="secondary">{t(checking ? 'checkingSession' : 'preparingWorkspace')}</Text>
      </VStack>
    </Center>
  );
}
