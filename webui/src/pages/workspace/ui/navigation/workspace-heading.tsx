// Purpose: Name the active workspace view in document and assistive navigation.
// Pattern: Metadata presentation over existing scoped query data.
// Usage: WorkspaceShell renders one heading; file documents replace the title with their path.
// Related: webui/src/pages/workspace/ui/files/file-document.tsx.
import { Heading } from '@astryxdesign/core/Heading';
import { useQuery } from '@tanstack/react-query';
import { useEffect } from 'react';
import { stateOptions } from '@/pages/workspace/api/queries';
import type { Target } from '@/pages/workspace/model/store';
import type { WorkspaceJson } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
export function WorkspaceHeading({
  target,
  mode,
  workspaces,
}: {
  target: Target;
  mode: 'chat' | 'files' | 'skills';
  workspaces: WorkspaceJson[];
}) {
  const t = useLocale();
  const state = useQuery({ ...stateOptions(target), enabled: !!target.session });
  const workspace = workspaces.find((value) => value.id === target.workspace)?.name ?? 'MECA';
  const label = mode === 'chat' ? state.data?.session?.summary || t('chat') : t(mode);
  const title = `${label} · ${workspace}`;
  useEffect(() => {
    document.title = `${title} — MECA`;
  }, [title]);
  return (
    <Heading level={1} className="sr-only">
      {title}
    </Heading>
  );
}
