// Purpose: Present team mode, follow-up reservation and active-task controls through Astryx.
// Pattern: Application action composition over controlled native operations.
// Usage: Composer supplies ChatComposer footer and send-action slots.
// Related: webui/src/pages/workspace/ui/composer/composer.tsx, webui/src/pages/workspace/model/execution.ts.

import { Icon } from '@astryxdesign/core/Icon';
import { IconButton } from '@astryxdesign/core/IconButton';
import { HStack } from '@astryxdesign/core/Layout';
import { ToggleButton } from '@astryxdesign/core/ToggleButton';
import { Tooltip } from '@astryxdesign/core/Tooltip';
import { reportError } from '@/entities/server-session';
import { team } from '@/pages/workspace/model/commands';
import type { Target } from '@/pages/workspace/model/store';
import { useLocale } from '@/shared/i18n';

export function ComposerActions({
  target,
  teamEnabled,
  busy,
}: {
  target: Target;
  teamEnabled: boolean;
  busy: boolean;
}) {
  const t = useLocale();
  return (
    <HStack gap={1}>
      <Tooltip content={t('team')}>
        <ToggleButton
          label={t('team')}
          isIconOnly
          icon={<Icon icon={'meca:users'} />}
          isPressed={teamEnabled}
          isDisabled={busy || !target.session}
          onPressedChange={(value) => {
            void team(target, value).catch(reportError);
          }}
        />
      </Tooltip>
    </HStack>
  );
}
export function ComposerQueueAction({
  disabled,
  submit,
}: {
  disabled: boolean;
  submit: (reserve?: boolean) => Promise<void>;
}) {
  const t = useLocale();
  return (
    <Tooltip content={t('queue')}>
      <IconButton
        label={t('queue')}
        variant="ghost"
        icon={<Icon icon="meca:queue" />}
        isDisabled={disabled}
        onClick={() => submit(true)}
      />
    </Tooltip>
  );
}
