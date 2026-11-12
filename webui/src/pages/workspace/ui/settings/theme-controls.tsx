// Purpose: Expose the persisted light/dark shortcut in the sidebar footer.
// Pattern: Controlled Astryx presentation controls.
// Usage: Sidebar uses the shared appearance preference for its theme action.
// Related: webui/src/pages/workspace/ui/navigation/sidebar.tsx, webui/src/shared/config/index.ts.

import { Icon } from '@astryxdesign/core/Icon';
import { IconButton } from '@astryxdesign/core/IconButton';
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
