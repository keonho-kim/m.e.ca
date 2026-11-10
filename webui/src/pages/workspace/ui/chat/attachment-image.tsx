// Purpose: Load small authenticated previews with bounded retries and release browser blob URLs.
// Pattern: Query-owned thumbnail with native Astryx image presentation.
// Usage: HistoryRow opens the original only when the user selects its preview.
// Related: webui/src/shared/api/client.ts, internal/serverapi/history_image.go.
import { Button } from '@astryxdesign/core/Button';
import { Thumbnail } from '@astryxdesign/core/Thumbnail';
import { useQuery } from '@tanstack/react-query';
import { memo, useEffect, useState } from 'react';
import { reportError } from '@/entities/server-session';
import type { Target } from '@/pages/workspace/model/store';
import { loadThumbnail } from '@/shared/api';
import { useLocale } from '@/shared/i18n';

export const AttachmentImage = memo(function AttachmentImage({
  target,
  sequence,
  index,
  onOpen,
}: {
  target: Target;
  sequence: string;
  index: number;
  onOpen: () => void;
}) {
  const t = useLocale(),
    [src, setSrc] = useState('');
  const query = useQuery({
    queryKey: ['thumbnail', target.workspace, target.session, sequence, index],
    queryFn: ({ signal }) =>
      loadThumbnail(target.workspace, target.session, sequence, index, signal),
    gcTime: 15000,
    staleTime: Infinity,
  });
  useEffect(() => {
    if (!query.data) return;
    const url = URL.createObjectURL(query.data);
    setSrc(url);
    return () => URL.revokeObjectURL(url);
  }, [query.data]);
  useEffect(() => {
    if (query.error) reportError(query.error);
  }, [query.error]);
  if (query.error) return <Button label={t('retry')} onClick={() => query.refetch()} />;
  return (
    <Thumbnail
      src={src || undefined}
      isLoading={!query.data || !src}
      alt={t('attachment')}
      onClick={src ? onOpen : undefined}
    />
  );
});
