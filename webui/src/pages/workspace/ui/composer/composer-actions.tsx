// Purpose: Compose the requested attachment expansion, team and follow-up controls through Astryx.
// Pattern: Application action composition over controlled native operations.
// Usage: Composer owns expansion and supplies ChatComposer footer/send slots.
// Related: webui/src/pages/workspace/ui/composer/composer.tsx, webui/src/pages/workspace/ui/composer/attachment-panel.tsx.

import { Icon } from '@astryxdesign/core/Icon';
import { IconButton } from '@astryxdesign/core/IconButton';
import { HStack } from '@astryxdesign/core/Layout';
import { ToggleButton } from '@astryxdesign/core/ToggleButton';
import { Tooltip } from '@astryxdesign/core/Tooltip';
import { reportError } from '@/entities/server-session';
import { team } from '@/pages/workspace/model/commands';
import type { Target } from '@/pages/workspace/model/store';
import { control } from '@/shared/api';
import { useLocale } from '@/shared/i18n';

export function ComposerActions({
  target,
  teamEnabled,
  busy,
  expanded,
  onExpandedChange,
  panelId,
}: {
  target: Target;
  teamEnabled: boolean;
  busy: boolean;
  expanded: boolean;
  onExpandedChange: (expanded: boolean) => void;
  panelId: string;
}) {
  const t = useLocale();
  return (
    <HStack gap={1}>
      <IconButton
        label={t('composerMenu')}
        variant="attachment"
        icon={<Icon icon={'meca:add'} />}
        aria-expanded={expanded}
        aria-controls={panelId}
        isDisabled={!target.session}
        onClick={() => onExpandedChange(!expanded)}
      />
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
    </HStack>
  );
}
export function ComposerSendActions({
  target,
  task,
  busy,
  submit,
}: {
  target: Target;
  task?: string;
  busy: boolean;
  submit: (reserve?: boolean) => Promise<void>;
}) {
  const t = useLocale();
  return (
    <HStack gap={1}>
      <Tooltip content={t('queue')}>
        <IconButton
          label={t('queue')}
          variant="ghost"
          icon={<Icon icon={'meca:queue'} />}
          isDisabled={busy || !target.session}
          onClick={() => submit(true)}
        />
      </Tooltip>
      {task && (
        <IconButton
          label={t('stop')}
          variant="ghost"
          icon={<Icon icon={'stop'} />}
          onClick={() => {
            void control(target.workspace, task, 'cancel').catch(reportError);
          }}
        />
      )}
    </HStack>
  );
}
