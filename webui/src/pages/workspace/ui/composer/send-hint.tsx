// Purpose: Explain send, steering and follow-up keys as separate platform-aware shortcut rows.
// Pattern: Read-only Astryx tooltip content.
// Usage: Composer mounts this inside the send button's tooltip.
// Related: webui/src/pages/workspace/ui/composer/composer.tsx, webui/src/shared/i18n/index.ts.
import { Kbd } from '@astryxdesign/core/Kbd';
import { HStack, VStack } from '@astryxdesign/core/Layout';
import { Text } from '@astryxdesign/core/Text';
import { useLocale } from '@/shared/i18n';

const shortcuts = [
  { label: 'sendIdle', keys: 'enter' },
  { label: 'steerActive', keys: 'enter' },
  { label: 'queueActive', keys: 'mod+enter' },
] as const;

export function ComposerSendHint() {
  const t = useLocale();
  return (
    <VStack gap={2} className="composer-send-hint">
      {shortcuts.map(({ label, keys }) => (
        <HStack
          key={label}
          gap={6}
          align="center"
          justify="between"
          className="composer-shortcut-row"
        >
          <Text type="inherit" color="inherit" className="whitespace-nowrap">
            {t(label)}
          </Text>
          <Kbd keys={keys} />
        </HStack>
      ))}
    </VStack>
  );
}
