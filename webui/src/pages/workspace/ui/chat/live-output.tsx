// Purpose: Render generation-scoped live responses and bounded task activity independently of history layout.
// Pattern: Fine-grained live presentation subscriber.
// Usage: Chat mounts this for its selected conversation.
// Related: webui/src/pages/workspace/model/stream-projection.ts, webui/src/pages/workspace/model/inspection.ts, webui/src/pages/workspace/ui/chat/live-activity.tsx.

import { VStack } from '@astryxdesign/core/Layout';
import { Selector } from '@astryxdesign/core/Selector';
import { useState } from 'react';
import { liveActivity } from '@/pages/workspace/model/chat-events';
import type { InspectionActions } from '@/pages/workspace/model/inspection';
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
    drafts = useWorkspaceStore((s) => s.drafts),
    events = useWorkspaceStore((s) => s.events);
  const [agent, setAgent] = useState('');
  const selected = useWorkspaceStore((s) => s.selection);
  const streaming = useWorkspaceStore((s) => s.streamStatus === 'connected');
  const sources =
    selected.workspace === target.workspace && selected.session === target.session
      ? Object.values(drafts)
      : [];
  const agents = [...new Set(sources.map((value) => value.agent).filter(Boolean))];
  const activity = liveActivity(events, history);
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
            streaming={streaming && d.doneRevision === undefined}
          />
        ))}
      <LiveActivity target={target} events={activity} inspect={inspect} />
    </VStack>
  );
}
