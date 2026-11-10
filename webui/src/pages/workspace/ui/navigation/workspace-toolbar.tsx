// Purpose: Separate work-area tabs from the current project and conversation context.
// Pattern: Two-row Astryx navigation composition.
// Usage: WorkspaceShell supplies this as the working area's Layout header.
// Related: webui/src/pages/workspace/model/commands.ts, webui/src/pages/workspace/ui/workspace-page.tsx.

import { useAppShellMobile } from '@astryxdesign/core/AppShell';
import { Button } from '@astryxdesign/core/Button';
import { Icon } from '@astryxdesign/core/Icon';
import { IconButton } from '@astryxdesign/core/IconButton';
import { HStack, VStack } from '@astryxdesign/core/Layout';
import {
  SideNavCollapseButton,
  type SideNavControlledCollapsible,
} from '@astryxdesign/core/SideNav';
import { Tab, TabList } from '@astryxdesign/core/TabList';
import { Text } from '@astryxdesign/core/Text';
import { Toolbar } from '@astryxdesign/core/Toolbar';
import { view } from '@/pages/workspace/model/commands';
import { useWorkspaceStore } from '@/pages/workspace/model/store';
import { useLocale } from '@/shared/i18n';
export function WorkspaceToolbar({
  name,
  summary,
  collapsible,
}: {
  name?: string;
  summary?: string;
  collapsible: SideNavControlledCollapsible;
}) {
  const t = useLocale(),
    mode = useWorkspaceStore((s) => s.mode),
    mobile = useAppShellMobile();
  return (
    <VStack gap={0} className="workspace-header min-w-0">
      <Toolbar
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
          <HStack gap={2}>
            <Button
              label={t('tasks')}
              variant="ghost"
              size="sm"
              icon={<Icon icon={'meca:layers'} />}
              onClick={() => useWorkspaceStore.getState().set({ tasksOpen: true })}
            />
            {collapsible.isCollapsed && <SideNavCollapseButton collapsible={collapsible} />}
          </HStack>
        }
      />
      <HStack
        paddingInline={4}
        paddingBlock={2}
        className="min-w-0"
        role="group"
        aria-label={t('workspaceContext')}
      >
        <Text
          type="label"
          weight="semibold"
          maxLines={2}
          hasTruncateTooltip
          className="min-w-0 break-words"
        >
          {summary || name || 'MECA'}
        </Text>
      </HStack>
    </VStack>
  );
}
