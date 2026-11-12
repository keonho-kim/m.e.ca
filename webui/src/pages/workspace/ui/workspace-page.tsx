// Purpose: Compose workspace navigation, live work areas and task controls after authentication.
// Pattern: Browser composition root.
// Usage: TanStack Router keeps App mounted while project/conversation routes change.
// Related: webui/src/pages/workspace/ui/navigation/workspace-toolbar.tsx, webui/src/pages/workspace/ui/navigation/sidebar.tsx, webui/src/pages/workspace/api/events.ts.

import { AlertDialog } from '@astryxdesign/core/AlertDialog';
import { AppShell } from '@astryxdesign/core/AppShell';
import { Banner } from '@astryxdesign/core/Banner';
import { Button } from '@astryxdesign/core/Button';
import { Center } from '@astryxdesign/core/Center';
import { ChatLayout } from '@astryxdesign/core/Chat';
import { EmptyState } from '@astryxdesign/core/EmptyState';
import { Layout, LayoutContent } from '@astryxdesign/core/Layout';
import type { SideNavControlledCollapsible } from '@astryxdesign/core/SideNav';
import { Spinner } from '@astryxdesign/core/Spinner';
import { useQuery } from '@tanstack/react-query';
import { useBlocker, useNavigate, useRouterState } from '@tanstack/react-router';
import {
  lazy,
  Suspense,
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import {
  dismissNotice,
  reportError,
  setServerLanguage,
  useServerSession,
} from '@/entities/server-session';
import { observeConversationCache } from '@/pages/workspace/api/cache-policy';
import { observe } from '@/pages/workspace/api/events';
import { capabilitiesOptions, changed, workspaceOptions } from '@/pages/workspace/api/queries';
import { type Target, useWorkspaceStore } from '@/pages/workspace/model/store';
import { useFileReferences } from '@/pages/workspace/model/use-file-references';
import { Composer, type ComposerHandle } from '@/pages/workspace/ui/composer/composer';
import { Approval } from '@/pages/workspace/ui/inspection/approval';
import { Sidebar } from '@/pages/workspace/ui/navigation/sidebar';
import { WorkspaceToolbar } from '@/pages/workspace/ui/navigation/workspace-toolbar';
import { call } from '@/shared/api';
import { useLocale } from '@/shared/i18n';

const Settings = lazy(() => import('@/pages/workspace/ui/settings/settings'));
const Chat = lazy(() =>
  import('@/pages/workspace/ui/chat/chat').then((module) => ({ default: module.Chat }))
);
const Files = lazy(() => import('@/pages/workspace/ui/files/files'));
const Skills = lazy(() => import('@/pages/workspace/ui/skills/skills'));
const FolderPicker = lazy(() => import('@/pages/workspace/ui/navigation/folder-picker'));

export default function WorkspaceShell() {
  const composerInput = useRef<ComposerHandle>(null);
  useEffect(observeConversationCache, []);
  const t = useLocale(),
    route = useNavigate();
  const params = useRouterState({ select: (state) => state.matches.at(-1)?.params }) as {
    workspaceId?: string;
    sessionId?: string;
  };
  const target = useMemo(
    () => ({ workspace: params.workspaceId ?? '', session: params.sessionId ?? '' }),
    [params.workspaceId, params.sessionId]
  );
  useFileReferences(target);
  const workspaces = useQuery(workspaceOptions());
  const review = useWorkspaceStore((s) => s.reviewTarget);
  const removed = useWorkspaceStore((s) => s.removedTarget);
  const approvalTarget = review ?? target;
  const closeReview = useCallback(
    () => useWorkspaceStore.getState().set({ reviewTarget: null }),
    []
  );
  const composerTarget = useMemo(
    () =>
      removed?.workspace === target.workspace && removed.session === target.session
        ? { workspace: target.workspace, session: '' }
        : target,
    [removed, target]
  );
  const mode = useWorkspaceStore((s) => s.mode),
    settings = useWorkspaceStore((s) => s.settings),
    picker = useWorkspaceStore((s) => s.picker);
  const notice = useServerSession((s) => s.notice);
  const connected = useServerSession((s) => s.connected);
  const capabilities = useQuery({
    ...capabilitiesOptions,
    enabled: connected,
  });
  useEffect(() => {
    if (capabilities.data?.language) setServerLanguage(capabilities.data.language);
  }, [capabilities.data?.language]);
  const fileDirty = useWorkspaceStore((s) => s.fileDirty);
  const [creating, setCreating] = useState(false);
  const [scroller, setScroller] = useState<HTMLDivElement | null>(null);
  const [landing, setLanding] = useState(false);
  const [isCollapsed, onCollapsedChange] = useState(false);
  const collapsible = useMemo<SideNavControlledCollapsible>(
    () => ({ isCollapsed, onCollapsedChange, hasButton: false }),
    [isCollapsed]
  );
  const navigation = useBlocker({
    shouldBlockFn: () => fileDirty,
    withResolver: true,
    enableBeforeUnload: fileDirty,
  });
  const navigate = useCallback(
    (next: Target) => {
      if (next.session)
        void route({
          to: '/w/$workspaceId/c/$sessionId',
          params: { workspaceId: next.workspace, sessionId: next.session },
        });
      else void route({ to: '/w/$workspaceId', params: { workspaceId: next.workspace } });
    },
    [route]
  );
  useEffect(() => {
    if (!target.workspace && workspaces.data?.defaultId)
      navigate({ workspace: workspaces.data.defaultId, session: '' });
  }, [target.workspace, workspaces.data?.defaultId, navigate]);
  useLayoutEffect(() => {
    useWorkspaceStore
      .getState()
      .set({ selection: target, drafts: {}, events: [], statistics: null, mode: 'chat' });
  }, [target]);
  useEffect(() => {
    if (!target.workspace || !connected) return;
    return observe(target);
  }, [target, connected]);
  useEffect(() => {
    if (removed?.workspace === target.workspace && removed.session === target.session && !fileDirty)
      navigate({ workspace: target.workspace, session: '' });
  }, [removed, target, fileDirty, navigate]);
  const create = async (workspace: string) => {
    if (creating || fileDirty) return;
    setCreating(true);
    try {
      const reply = await call(workspace, { selectSession: { create: true } });
      const id = reply.state?.session?.id;
      if (id) {
        changed({ workspace, session: id });
        navigate({ workspace, session: id });
      }
    } catch (error) {
      reportError(error);
    } finally {
      setCreating(false);
    }
  };
  const composer = (
    <Composer
      ref={composerInput}
      key={`${composerTarget.workspace}/${composerTarget.session}`}
      target={composerTarget}
      showStatus={mode === 'chat'}
    />
  );
  return (
    <AppShell
      variant="surface"
      height="fill"
      contentPadding={0}
      sideNav={
        <Sidebar
          workspaces={workspaces.data?.items ?? []}
          target={target}
          navigate={navigate}
          create={create}
          collapsible={collapsible}
        />
      }
      mobileNav={{ hasToggle: false }}
    >
      <Layout
        header={<WorkspaceToolbar />}
        height="fill"
        padding={0}
        content={
          <LayoutContent
            padding={0}
            id="meca-work-area"
            role="tabpanel"
            aria-labelledby={`meca-tab-${mode}`}
          >
            {notice && (
              <Banner status="error" title={notice} isDismissable onDismiss={dismissNotice} />
            )}
            {workspaces.error && <Button label={t('retry')} onClick={() => workspaces.refetch()} />}
            {mode === 'chat' ? (
              <Center height="100%" width="100%">
                <ChatLayout
                  key={`${target.workspace}/${target.session}`}
                  ref={setScroller}
                  className="workspace-body"
                  density="balanced"
                  composer={composer}
                  style={{ height: landing ? 'auto' : '100%', width: '100%', minWidth: 0 }}
                >
                  <Suspense
                    fallback={
                      <Center padding={6}>
                        <Spinner label={t('loading')} />
                      </Center>
                    }
                  >
                    {target.session ? (
                      <Chat target={target} scroller={scroller} onEmpty={setLanding} />
                    ) : (
                      <Center padding={8}>
                        <EmptyState
                          title={t('emptyTitle')}
                          description={t('emptyHelp')}
                          actions={
                            <Button
                              label={t('newChat')}
                              variant="primary"
                              isDisabled={!target.workspace}
                              isLoading={creating}
                              onClick={() => create(target.workspace)}
                            />
                          }
                        />
                      </Center>
                    )}
                  </Suspense>
                </ChatLayout>
              </Center>
            ) : (
              <Layout
                height="fill"
                padding={0}
                content={
                  <LayoutContent padding={0}>
                    <Suspense
                      fallback={
                        <Center padding={6}>
                          <Spinner label={t('loading')} />
                        </Center>
                      }
                    >
                      {mode === 'files' ? (
                        <Files target={target} />
                      ) : (
                        <Skills
                          target={target}
                          onInsert={(mention) => composerInput.current?.insertSkill(mention)}
                        />
                      )}
                    </Suspense>
                  </LayoutContent>
                }
                footer={composer}
              />
            )}
          </LayoutContent>
        }
      />
      <Suspense fallback={null}>
        {settings && <Settings />}
        {picker && <FolderPicker navigate={navigate} />}
      </Suspense>
      {approvalTarget.session && (
        <Approval
          key={`${approvalTarget.workspace}/${approvalTarget.session}`}
          target={approvalTarget}
          onResolved={closeReview}
        />
      )}
      {navigation.status === 'blocked' && (
        <AlertDialog
          isOpen
          title={t('dirty')}
          description={t('unsavedFile')}
          cancelLabel={t('keepEditing')}
          actionLabel={t('discard')}
          onOpenChange={(open) => {
            if (!open) navigation.reset();
          }}
          onAction={() => navigation.proceed()}
        />
      )}
    </AppShell>
  );
}
