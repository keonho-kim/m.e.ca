// Purpose: Render generation-scoped live responses and bounded task activity independently of history layout.
// Pattern: Fine-grained live presentation subscriber.
// Usage: Chat mounts this for its selected conversation.
// Related: webui/src/pages/workspace/model/stream-projection.ts, webui/src/pages/workspace/model/inspection.ts, webui/src/pages/workspace/ui/chat/live-activity.tsx.

import { VStack } from '@astryxdesign/core/Layout';
import { Selector } from '@astryxdesign/core/Selector';
import { useMemo, useState, useSyncExternalStore } from 'react';
import { liveActivity } from '@/pages/workspace/model/chat-events';
import type { InspectionActions } from '@/pages/workspace/model/inspection';
import { useChatPresentationBlocked } from '@/pages/workspace/model/presentation';
import { type Target, useWorkspaceStore } from '@/pages/workspace/model/store';
import { LiveActivity } from '@/pages/workspace/ui/chat/live-activity';
import { LiveSource } from '@/pages/workspace/ui/chat/live-source';
import type { HistoryItemJson } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
export function LiveOutput({
  target,
  history,
  inspect,
}: {
  target: Target;
  history: HistoryItemJson[];
  inspect: InspectionActions;
}) {
  const t = useLocale(),
    visible = useVisibleOutput();
  const { drafts, events } = visible;
  const [agent, setAgent] = useState('');
  const selected = useWorkspaceStore((s) => s.selection);
  const blocked = useChatPresentationBlocked();
  const streaming = useWorkspaceStore((s) => s.streamStatus === 'connected');
  const sources =
    selected.workspace === target.workspace && selected.session === target.session
      ? Object.values(drafts)
      : [];
  const agents = [...new Set(sources.map((value) => value.agent).filter(Boolean))];
  const activity = useMemo(() => liveActivity(events, history), [events, history]);
  if (!sources.length && !events.length) return null;
  return (
    <VStack width="100%" className="live-output" gap={8} paddingBlockStart={8}>
      {agents.length > 1 && (
        <Selector
          label={t('agent')}
          value={agent}
          onChange={setAgent}
          options={[
            { value: '', label: t('allAgents') },
            ...agents.map((value) => ({ value, label: value })),
          ]}
        />
      )}
      {sources
        .filter((v) => (v.text || v.thinking) && (!agent || v.agent === agent))
        .map((d) => (
          <LiveSource
            key={`${d.task}/${d.agent}/${d.generation}`}
            draft={d}
            streaming={!blocked && streaming && d.doneRevision === undefined}
          />
        ))}
      <LiveActivity target={target} events={activity} inspect={inspect} />
    </VStack>
  );
}

// Keep one immutable projection while covered. The canonical store continues
// processing approvals, completion and queues; no copied transcript is retained.
function useVisibleOutput() {
  const blocked = useChatPresentationBlocked();
  const [owner] = useState(() => {
    const initial = useWorkspaceStore.getState();
    let previous = { drafts: initial.drafts, events: initial.events };
    return {
      read: (paused: boolean) => {
        const next = useWorkspaceStore.getState();
        if (
          (!paused || (!Object.keys(next.drafts).length && !next.events.length)) &&
          (next.drafts !== previous.drafts || next.events !== previous.events)
        )
          previous = { drafts: next.drafts, events: next.events };
        return previous;
      },
    };
  });
  return useSyncExternalStore(useWorkspaceStore.subscribe, () => owner.read(blocked));
}
