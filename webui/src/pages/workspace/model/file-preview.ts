// Purpose: Own revision-bound navigation through read-only file pages.
// Pattern: Query-backed paging hook.
// Usage: FileDocument renders content and navigation in separate layout slots.
// Related: webui/src/pages/workspace/ui/files/file-preview.tsx, webui/src/shared/api/index.ts.
import { useQuery } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import { call } from '@/shared/api';
import { failure } from '@/shared/lib/errors';

export function useFilePreview(workspace: string, path: string, enabled: boolean, version: number) {
  const [offsets, setOffsets] = useState([0]),
    [revision, setRevision] = useState('');
  useEffect(() => {
    setOffsets([0]);
    setRevision('');
  }, [version]);
  const offset = offsets.at(-1) ?? 0;
  const query = useQuery({
    queryKey: ['preview', workspace, path, offset, revision],
    queryFn: async ({ signal }) => {
      const reply = await call(
        workspace,
        { previewFile: { path, offset, limit: 65536, ...(offset ? { revision } : {}) } },
        signal
      );
      if (!reply.filePreview) throw failure('invalid_response');
      return reply.filePreview;
    },
    enabled,
    staleTime: 0,
    gcTime: 0,
  });
  const hasNext =
    !!query.data?.text && Number(query.data.nextOffset ?? 0) < Number(query.data.totalBytes ?? 0);
  return {
    data: query.data,
    pending: query.isPending,
    error: query.error,
    retry: query.refetch,
    offset,
    hasNext,
    hasPrevious: offsets.length > 1,
    previous: () => setOffsets((value) => (value.length > 1 ? value.slice(0, -1) : value)),
    next: () => {
      if (!hasNext) return;
      setRevision(query.data?.revision ?? '');
      setOffsets((value) => [...value, Number(query.data?.nextOffset ?? 0)]);
    },
  };
}
