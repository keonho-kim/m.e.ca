// Purpose: Compose Astryx navigation for registered projects and their paged conversations.
// Pattern: Application navigation composition with library-owned responsive behavior.
// Usage: WorkspaceShell supplies this to AppShell's sideNav slot.
// Related: webui/src/pages/workspace/ui/navigation/workspace-group.tsx, webui/src/pages/workspace/ui/settings/theme-controls.tsx.

import { useAppShellMobile } from '@astryxdesign/core/AppShell';
import { Button } from '@astryxdesign/core/Button';
import { Icon } from '@astryxdesign/core/Icon';
import { IconButton } from '@astryxdesign/core/IconButton';
import { HStack, VStack } from '@astryxdesign/core/Layout';
import {
  SideNav,
  SideNavCollapseButton,
  type SideNavControlledCollapsible,
  SideNavHeading,
} from '@astryxdesign/core/SideNav';
import { type Target, useWorkspaceStore } from '@/pages/workspace/model/store';
import { WorkspaceGroup } from '@/pages/workspace/ui/navigation/workspace-group';
import { ThemeShortcut } from '@/pages/workspace/ui/settings/theme-controls';
import type { WorkspaceJson } from '@/shared/api';
import { useLocale } from '@/shared/i18n';

export function Sidebar({
  workspaces,
  target,
  navigate,
  create,
  collapsible,
}: {
  workspaces: WorkspaceJson[];
  target: Target;
  navigate: (target: Target) => void;
  create: (workspace: string) => Promise<void>;
  collapsible: SideNavControlledCollapsible;
}) {
  const t = useLocale(),
    mobile = useAppShellMobile();
  return (
    <SideNav
      id="workspace-navigation"
      aria-label={t('workspaceList')}
      resizable
      collapsible={collapsible}
      header={
        <SideNavHeading
          heading="MECA"
          headerEndContent={
            <HStack gap={1}>
              <IconButton
                label={t('addWorkspace')}
                variant="ghost"
                icon={<Icon icon={'meca:folder-add'} />}
                onClick={() => {
                  mobile.closeMobileNav();
                  useWorkspaceStore.getState().set({ picker: true });
                }}
              />
              {!collapsible.isCollapsed && <SideNavCollapseButton />}
            </HStack>
          }
        />
      }
      footer={
        <VStack gap={2}>
          <HStack justify="between" wrap="wrap">
            <Button
              isIconOnly={collapsible.isCollapsed}
              label={t('settings')}
              variant="ghost"
              icon={<Icon icon={'meca:settings'} />}
              onClick={() => {
                mobile.closeMobileNav();
                useWorkspaceStore.getState().set({ settings: { tab: 'model', target } });
              }}
            />
            <ThemeShortcut />
          </HStack>
        </VStack>
      }
    >
      <VStack gap={4} paddingBlock={3}>
        {workspaces.map((workspace) => (
          <WorkspaceGroup
            key={workspace.id}
            workspace={workspace}
            target={target}
            navigate={(next) => {
              mobile.closeMobileNav();
              navigate(next);
            }}
            create={async (id) => {
              await create(id);
              mobile.closeMobileNav();
            }}
          />
        ))}
      </VStack>
    </SideNav>
  );
}
