// Purpose: Present work-area tabs, mobile navigation and the global language menu.
// Pattern: Astryx toolbar composition.
// Usage: WorkspaceShell supplies this as the working area's Layout header.
// Related: webui/src/pages/workspace/model/commands.ts, webui/src/pages/workspace/ui/workspace-page.tsx.

import { useAppShellMobile } from '@astryxdesign/core/AppShell';
import { Icon } from '@astryxdesign/core/Icon';
import { IconButton } from '@astryxdesign/core/IconButton';
import { Tab, TabList } from '@astryxdesign/core/TabList';
import { Toolbar } from '@astryxdesign/core/Toolbar';
import { view } from '@/pages/workspace/model/commands';
import { useWorkspaceStore } from '@/pages/workspace/model/store';
import { LanguageMenu } from '@/pages/workspace/ui/navigation/language-menu';
import { useLocale } from '@/shared/i18n';
export function WorkspaceToolbar() {
  const t = useLocale(),
    mode = useWorkspaceStore((s) => s.mode),
    mobile = useAppShellMobile();
  return (
    <Toolbar
      className="workspace-header min-w-0"
      label={t('workspaceView')}
      size="lg"
      startContent={
        mobile.isMobile ? (
          <IconButton
            label={t('open')}
            variant="ghost"
            icon={<Icon icon={'meca:sidebar'} />}
            aria-expanded={mobile.isMobileNavOpen}
            aria-controls={mobile.mobileNavId}
            onClick={mobile.toggleMobileNav}
          />
        ) : undefined
      }
      endContent={<LanguageMenu />}
      centerContent={
        <TabList
          value={mode}
          role="tablist"
          aria-label={t('workspaceView')}
          size="md"
          onChange={(value) => {
            if (value === 'chat' || value === 'files' || value === 'skills') view(value);
          }}
        >
          <Tab
            value="chat"
            id="meca-tab-chat"
            label={t('chat')}
            icon={<Icon icon={'meca:message'} />}
            panelId="meca-work-area"
            isLabelHidden={mobile.isMobile}
          />
          <Tab
            value="files"
            id="meca-tab-files"
            label={t('files')}
            icon={<Icon icon={'meca:folder-open'} />}
            panelId="meca-work-area"
            isLabelHidden={mobile.isMobile}
          />
          <Tab
            value="skills"
            id="meca-tab-skills"
            label={t('skills')}
            icon={<Icon icon={'meca:book'} />}
            panelId="meca-work-area"
            isLabelHidden={mobile.isMobile}
          />
        </TabList>
      }
    />
  );
}
