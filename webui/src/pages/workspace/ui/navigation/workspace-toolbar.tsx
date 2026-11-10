// Purpose: Compose project context and work-area navigation in an official Astryx toolbar.
// Pattern: Controlled application navigation.
// Usage: WorkspaceShell supplies this as the working area's Layout header.
// Related: webui/src/pages/workspace/model/commands.ts, webui/src/pages/workspace/ui/workspace-page.tsx.

import { useAppShellMobile } from '@astryxdesign/core/AppShell';
import { Button } from '@astryxdesign/core/Button';
import { Icon } from '@astryxdesign/core/Icon';
import { IconButton } from '@astryxdesign/core/IconButton';
import { HStack } from '@astryxdesign/core/Layout';
import { Tab, TabList } from '@astryxdesign/core/TabList';
import { Text } from '@astryxdesign/core/Text';
import { Toolbar } from '@astryxdesign/core/Toolbar';
import { view } from '@/pages/workspace/model/commands';
import { useWorkspaceStore } from '@/pages/workspace/model/store';
import { useLocale } from '@/shared/i18n';
export function WorkspaceToolbar({ name, summary }: { name?: string; summary?: string }) {
  const t = useLocale(),
    mode = useWorkspaceStore((s) => s.mode),
    mobile = useAppShellMobile();
  return (
    <Toolbar
      label={t('workspaceView')}
      className="workspace-header"
      size="lg"
      startContent={
        <HStack gap={3}>
          {mobile.isMobile && (
            <IconButton
              label={t('open')}
              variant="ghost"
              icon={<Icon icon={'meca:sidebar'} />}
              aria-expanded={mobile.isMobileNavOpen}
              aria-controls={mobile.mobileNavId}
              onClick={mobile.toggleMobileNav}
            />
          )}
          <Text type="label" weight="semibold" maxLines={1}>
            {summary || name || 'MECA'}
          </Text>
        </HStack>
      }
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
      endContent={
        <Button
          label={t('tasks')}
          variant="ghost"
          size="sm"
          icon={<Icon icon={'meca:layers'} />}
          onClick={() => useWorkspaceStore.getState().set({ tasksOpen: true })}
        />
      }
    />
  );
}
