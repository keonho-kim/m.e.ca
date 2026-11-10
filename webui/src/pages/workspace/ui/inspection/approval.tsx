// Purpose: Resolve the selected conversation's actual approvals with task-bound controls.
// Pattern: Explicit approval dialog.
// Usage: App renders this over the current task and restores focus after resolution.
// Related: webui/src/pages/workspace/ui/inspection/original.tsx, internal/commands/server/controls.go.

import { Button } from '@astryxdesign/core/Button';
import { Code } from '@astryxdesign/core/Code';
import { Dialog, DialogHeader } from '@astryxdesign/core/Dialog';
import { Heading } from '@astryxdesign/core/Heading';
import { HStack, Layout, LayoutContent, VStack } from '@astryxdesign/core/Layout';
import { Text } from '@astryxdesign/core/Text';
import { useQuery } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import { reportError } from '@/entities/server-session';
import { changed, stateOptions } from '@/pages/workspace/api/queries';
import { type Target, useWorkspaceStore } from '@/pages/workspace/model/store';
import { Original } from '@/pages/workspace/ui/inspection/original';
import { control } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { ApiError } from '@/shared/lib/errors';
import { CodePreview } from '@/shared/ui/code-preview';
export function Approval({ target, onResolved }: { target: Target; onResolved: () => void }) {
  const t = useLocale(),
    events = useWorkspaceStore((s) => s.events);
  const state = useQuery({
    ...stateOptions(target),
    enabled: !!target.session,
    refetchOnMount: 'always',
  });
  const [busy, setBusy] = useState(false),
    [full, setFull] = useState(false),
    [resolved, setResolved] = useState<string[]>([]);
  const relevant = events.filter(
    (e) => e.workspaceId === target.workspace && e.sessionId === target.session
  );
  const closed = new Set([
    ...resolved,
    ...relevant.filter((e) => e.kind === 'approval_closed').map((e) => e.approvalId),
  ]);
  const approvals = [
    ...(state.data?.approvals ?? []),
    ...relevant.flatMap((e) => (e.approval ? [e.approval] : [])),
  ];
  const approval = approvals.find((a) => a.id && !closed.has(a.id));
  useEffect(() => {
    if (!approval && !state.isFetching && state.isSuccess) onResolved();
  }, [approval, state.isFetching, state.isSuccess, onResolved]);
  if (!approval?.id || !approval.taskId) return null;
  const decide = async (allow: boolean) => {
    setBusy(true);
    try {
      await control(target.workspace, approval.taskId!, 'approve', allow, approval.id);
      setResolved((values) => [...values.slice(-127), approval.id!]);
      changed(target);
      onResolved();
    } catch (error) {
      if (
        error instanceof ApiError &&
        ['not_found', 'conflict', 'failed_precondition'].includes(error.code)
      ) {
        setResolved((values) => [...values.slice(-127), approval.id!]);
        onResolved();
      }
      reportError(error);
      changed(target);
    } finally {
      setBusy(false);
    }
  };
  return (
    <Dialog
      isOpen={true}
      onOpenChange={(open) => {
        if (!open && !busy) void decide(false);
      }}
      purpose="form"
      width={760}
      className="approval-modal"
    >
      <Layout
        header={
          <DialogHeader
            title={t('approval')}
            subtitle={state.data?.session?.displayRoot}
            onOpenChange={(open) => {
              if (!open && !busy) void decide(false);
            }}
          />
        }
        content={
          <LayoutContent padding={4}>
            <VStack gap={4}>
              <VStack className="approval-content" gap={4}>
                <Heading level={3} type="display-3">
                  {approval.name}
                </Heading>
                <Code>{approval.target}</Code>
                <CodePreview code={approval.preview ?? ''} language="plaintext" isWrapped={true} />
                {approval.expiresAt && (
                  <Text className="muted" as="p">
                    {t('expires')}: {new Date(approval.expiresAt).toLocaleString()}
                  </Text>
                )}
                {approval.truncated && (
                  <Button label={t('moreOriginal')} onClick={() => setFull(true)} />
                )}
              </VStack>
              <HStack className="form-actions" gap={2}>
                <Button label={t('deny')} isDisabled={busy} onClick={() => decide(false)} />
                <Button
                  label={t('approve')}
                  variant="primary"
                  isLoading={busy}
                  onClick={() => decide(true)}
                />
              </HStack>
              {full && (
                <Original
                  target={target}
                  source={{ approvalId: approval.id }}
                  onClose={() => setFull(false)}
                />
              )}
            </VStack>
          </LayoutContent>
        }
      />
    </Dialog>
  );
}
