// Purpose: Own one project's paged conversations and confirmed deletion through Astryx navigation.
// Pattern: Scoped query controller composed with official navigation and dialogs.
// Usage: Sidebar renders one WorkspaceGroup per project.
// Related: webui/src/pages/workspace/ui/navigation/sidebar.tsx, webui/src/pages/workspace/model/deletion.ts.

import { Badge } from '@astryxdesign/core/Badge';
import { Banner } from '@astryxdesign/core/Banner';
import { Button } from '@astryxdesign/core/Button';
import { Dialog, DialogHeader } from '@astryxdesign/core/Dialog';
import { Icon } from '@astryxdesign/core/Icon';
import { HStack, Layout, LayoutContent, VStack } from '@astryxdesign/core/Layout';
import { SideNavItem } from '@astryxdesign/core/SideNav';
import { Text } from '@astryxdesign/core/Text';
import { useEffect, useState } from 'react';
import { reportError } from '@/entities/server-session';
import { changed } from '@/pages/workspace/api/queries';
import { forgetConversation } from '@/pages/workspace/model/deletion';
import { type Target, useWorkspaceStore } from '@/pages/workspace/model/store';
import { workspaceTasks } from '@/pages/workspace/model/task-state';
import { WorkspaceSessions } from '@/pages/workspace/ui/navigation/workspace-sessions';
import type { SessionJson, WorkspaceJson } from '@/shared/api';
import { call } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { describeError, failure } from '@/shared/lib/errors';

export function WorkspaceGroup({
  workspace,
  target,
  navigate,
  create,
}: {
  workspace: WorkspaceJson;
  target: Target;
  navigate: (target: Target) => void;
  create: (id: string) => Promise<void>;
}) {
  const t = useLocale(),
    id = workspace.id ?? '';
  const [expanded, setExpanded] = useState(id === target.workspace),
    [deleting, setDeleting] = useState<SessionJson | null>(null);
  const [busy, setBusy] = useState(false),
    [error, setError] = useState('');
  const running = useWorkspaceStore((s) => workspaceTasks(id, s.tasks));
  useEffect(() => {
    if (id === target.workspace) setExpanded(true);
  }, [id, target.workspace]);
  const active = Math.max(workspace.activeTasks ?? 0, running.length);
  const remove = async () => {
    if (!deleting) return;
    if (
      useWorkspaceStore.getState().fileDirty &&
      target.workspace === id &&
      target.session === deleting.id
    ) {
      setError(describeError(failure('draft_changed')));
      return;
    }
    setBusy(true);
    try {
      await call(id, { deleteSession: { sessionId: deleting.id } });
      forgetConversation({ workspace: id, session: deleting.id ?? '' });
      changed({ workspace: id, session: deleting.id ?? '' });
      if (target.session === deleting.id) navigate({ workspace: id, session: '' });
      setDeleting(null);
    } catch (error) {
      setError(describeError(error));
      reportError(error);
    } finally {
      setBusy(false);
    }
  };
  return (
    <VStack className="workspace-group" gap={1}>
      <SideNavItem
        label={workspace.name ?? ''}
        className="workspace-name"
        icon={<Icon icon={'meca:folder'} />}
        collapsible={{
          isCollapsed: !expanded,
          onCollapsedChange: (collapsed) => setExpanded(!collapsed),
        }}
        isSelected={target.workspace === id && !target.session}
        endContent={active > 0 ? <Badge label={active} /> : undefined}
        onClick={() => {
          setExpanded(true);
          navigate({ workspace: id, session: '' });
        }}
      >
        {/* Long labels must not determine the collapsible grid's inline size. */}
        <VStack gap={1} style={{ contain: 'inline-size' }}>
          {expanded && (
            <>
              <Button
                label={t('newChat')}
                size="sm"
                variant="ghost"
                icon={<Icon icon={'meca:add'} />}
                onClick={() => create(id)}
              />
              <WorkspaceSessions
                workspace={id}
                target={target}
                navigate={navigate}
                remove={(session) => {
                  setError('');
                  setDeleting(session);
                }}
              />
              {workspace.error && <Banner status="error" title={workspace.error} />}
            </>
          )}
        </VStack>
      </SideNavItem>
      {deleting && (
        <Dialog
          isOpen
          onOpenChange={(open) => {
            if (!open && !busy) setDeleting(null);
          }}
          width={480}
          purpose="form"
          className="confirm-modal"
        >
          <Layout
            header={
              <DialogHeader
                title={t('deleteChat')}
                onOpenChange={() => {
                  if (!busy) setDeleting(null);
                }}
              />
            }
            content={
              <LayoutContent>
                <VStack gap={4}>
                  <Text>{t('deleteConfirm')}</Text>
                  <Text color="secondary">{deleting.summary || t('untitled')}</Text>
                  {error && <Banner status="error" title={error} />}
                </VStack>
              </LayoutContent>
            }
            footer={
              <HStack justify="end" gap={2} padding={4}>
                <Button label={t('cancel')} isDisabled={busy} onClick={() => setDeleting(null)} />
                <Button
                  label={t('deleteChat')}
                  variant="destructive"
                  isLoading={busy}
                  onClick={remove}
                />
              </HStack>
            }
          />
        </Dialog>
      )}
    </VStack>
  );
}
