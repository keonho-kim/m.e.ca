// Purpose: Expose persisted appearance choices and a direct light/dark shortcut.
// Pattern: Controlled Astryx presentation controls.
// Usage: WorkspaceToolbar and Settings share the same Zustand theme preference.
// Related: webui/src/pages/workspace/model/store.ts, webui/src/app/main.tsx.

import { Icon } from '@astryxdesign/core/Icon';
import { IconButton } from '@astryxdesign/core/IconButton';
import { SegmentedControl, SegmentedControlItem } from '@astryxdesign/core/SegmentedControl';
import { Tooltip } from '@astryxdesign/core/Tooltip';
import { useAppearance, useResolvedTheme } from '@/shared/config';
import { useLocale } from '@/shared/i18n';

export function ThemeShortcut() {
  const t = useLocale(),
    preference = useAppearance((s) => s.theme),
    mode = useResolvedTheme(preference);
  return (
    <Tooltip content={t('theme')} placement="below">
      <IconButton
        label={t('theme')}
        variant="ghost"
        className="theme-shortcut"
        icon={<Icon icon={mode === 'dark' ? 'meca:sun' : 'meca:moon'} />}
        onClick={() =>
          useAppearance
            .getState()
            .setTheme(document.documentElement.dataset.theme === 'dark' ? 'light' : 'dark')
        }
      />
    </Tooltip>
  );
}

export function ThemeChoice() {
  const t = useLocale(),
    theme = useAppearance((s) => s.theme);
  return (
    <SegmentedControl
      label={t('theme')}
      value={theme}
      onChange={(value) => {
        if (value === 'light' || value === 'dark' || value === 'system')
          useAppearance.getState().setTheme(value);
      }}
      layout="fill"
      className="theme-choice"
    >
      <SegmentedControlItem
        className="theme-option"
        value="light"
        label={t('light')}
        icon={<Icon icon={'meca:sun'} size="sm" />}
      />
      <SegmentedControlItem
        className="theme-option"
        value="dark"
        label={t('dark')}
        icon={<Icon icon={'meca:moon'} size="sm" />}
      />
      <SegmentedControlItem
        className="theme-option"
        value="system"
        label={t('themeSystem')}
        icon={<Icon icon={'meca:monitor'} size="sm" />}
      />
    </SegmentedControl>
  );
}
