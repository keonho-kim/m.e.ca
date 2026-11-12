// Purpose: Change the global interface language from a compact work-area toolbar menu.
// Pattern: Astryx action menu with one owned settings write.
// Usage: WorkspaceToolbar and !language open this same control.
// Related: webui/src/pages/workspace/model/commands.ts, webui/src/entities/server-session/model/session.ts.
import { DropdownMenu, DropdownMenuItem } from '@astryxdesign/core/DropdownMenu';
import { Icon } from '@astryxdesign/core/Icon';
import { IconButton } from '@astryxdesign/core/IconButton';
import { Tooltip } from '@astryxdesign/core/Tooltip';
import { useRef, useState } from 'react';
import { reportError, setServerLanguage, useServerSession } from '@/entities/server-session';
import { useWorkspaceStore } from '@/pages/workspace/model/store';
import { call, queries } from '@/shared/api';
import { useLocale } from '@/shared/i18n';

export function LanguageMenu() {
  const t = useLocale();
  const language = useServerSession((state) => state.language);
  const open = useWorkspaceStore((state) => state.languageMenu);
  const target = useWorkspaceStore((state) => state.selection);
  const pending = useRef(false);
  const [busy, setBusy] = useState(false);
  const choose = async (value: string) => {
    if (pending.current || !['auto', 'ko', 'en'].includes(value) || value === language) return;
    pending.current = true;
    setBusy(true);
    try {
      await call(target.workspace, { setSettings: { language: value } });
      setServerLanguage(value);
      void queries.invalidateQueries({ queryKey: ['capabilities'] });
    } catch (error) {
      reportError(error);
    } finally {
      pending.current = false;
      setBusy(false);
    }
  };
  return (
    <DropdownMenu
      presentation="popover"
      alignment="end"
      menuWidth={192}
      isMenuOpen={open}
      onOpenChange={(value) => useWorkspaceStore.getState().set({ languageMenu: value })}
      renderTrigger={(props) => (
        <Tooltip content={t('language')}>
          <IconButton
            {...props}
            label={t('language')}
            variant="ghost"
            icon={<Icon icon="meca:language" />}
          />
        </Tooltip>
      )}
    >
      {[
        { value: 'auto', label: t('system') },
        { value: 'ko', label: '한국어' },
        { value: 'en', label: 'English' },
      ].map(({ value, label }) => (
        <DropdownMenuItem
          key={value}
          label={label}
          aria-current={value === language ? 'true' : undefined}
          endContent={value === language ? <Icon icon="check" size="sm" /> : undefined}
          isDisabled={busy}
          onClick={() => {
            void choose(value);
          }}
        />
      ))}
    </DropdownMenu>
  );
}
