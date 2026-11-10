// Purpose: Render generation-scoped live responses and bounded task activity independently of history layout.
// Pattern: Fine-grained live presentation subscriber.
// Usage: Chat mounts this for its selected conversation.
// Related: webui/src/pages/workspace/model/stream-projection.ts, webui/src/pages/workspace/ui/inspection/original.tsx, webui/src/pages/workspace/ui/chat/live-activity.tsx.

import { VStack } from '@astryxdesign/core/Layout';
import { Selector } from '@astryxdesign/core/Selector';
import { useMemo, useState } from 'react';
import { type Target, useWorkspaceStore } from '@/pages/workspace/model/store';
import { LiveActivity } from '@/pages/workspace/ui/chat/live-activity';
import { LiveSource } from '@/pages/workspace/ui/chat/live-source';
import { Original } from '@/pages/workspace/ui/inspection/original';
import type { DraftRequestJson } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
export function LiveOutput({ target }: { target: Target }) {
  const t = useLocale(),
    drafts = useWorkspaceStore((s) => s.drafts),
    events = useWorkspaceStore((s) => s.events);
  const [agent, setAgent] = useState('');
  const [original, setOriginal] = useState<DraftRequestJson | null>(null);
  const selected = useWorkspaceStore((s) => s.selection);
  const sources = useMemo(
    () =>
      selected.workspace === target.workspace && selected.session === target.session
        ? Object.values(drafts)
        : [],
    [drafts, selected, target]
  );
  const agents = useMemo(
    () => [...new Set(sources.map((v) => v.agent).filter(Boolean))],
    [sources]
  );
  return (
    <VStack className="live-output" gap={5}>
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
        .filter((v) => !agent || v.agent === agent)
        .map((d) => (
          <LiveSource
            key={`${d.task}/${d.agent}/${d.generation}`}
            draft={d}
            onOriginal={setOriginal}
          />
        ))}
      <LiveActivity target={target} events={events} />
      {original && <Original target={target} draft={original} onClose={() => setOriginal(null)} />}
    </VStack>
  );
}
