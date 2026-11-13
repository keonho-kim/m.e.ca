// Purpose: Own revision-bound native detail pages without retaining complete tool or response documents.
// Pattern: Cancellable query lifecycle with explicit page navigation.
// Usage: Inline tool results, nested requests and long saved responses share these bounded reads.
// Related: webui/src/pages/workspace/ui/chat/tool-call.tsx, webui/src/pages/workspace/ui/chat/tool-request.tsx.
import { useQuery } from '@tanstack/react-query';
import { useState } from 'react';
import type { Target } from '@/pages/workspace/model/store';
import { call, type DetailRequestJson } from '@/shared/api';
import { ApiError, failure } from '@/shared/lib/errors';
export function useDetailPage(target: Target, source: DetailRequestJson) {
  const [positions, setPositions] = useState(
      source.offset && source.offset !== '0' ? ['0', source.offset] : ['0']
    ),
    [revision, setRevision] = useState(source.revision ?? '');
  const offset = positions.at(-1) ?? '0';
  const page = useQuery({
    queryKey: ['detail-page', target.workspace, target.session, source, offset, revision],
    gcTime: 0,
    queryFn: async ({ signal }) => {
      const reply = await call(
        target.workspace,
        {
          detail: {
            ...source,
            sessionId: target.session,
            offset,
            limit: 32768,
            revision: offset === '0' ? undefined : revision,
          },
        },
        signal
      );
      if (!reply.detail) throw failure('invalid_response');
      return reply.detail;
    },
  });
  return {
    page,
    offset,
    retry: () => {
      if (page.error instanceof ApiError && page.error.code === 'conflict' && offset !== '0') {
        setPositions(['0']);
        setRevision('');
      } else void page.refetch();
    },
    previous: () => {
      setPositions((old) => (old.length > 1 ? old.slice(0, -1) : old));
    },
    next: () => {
      const next = page.data?.nextOffset;
      if (next) {
        setRevision(page.data?.revision ?? '');
        setPositions((old) => [...old, next]);
      }
    },
    more: !!page.data && Number(page.data.nextOffset) < Number(page.data.totalBytes),
    hasPrevious: offset !== '0',
  };
}
