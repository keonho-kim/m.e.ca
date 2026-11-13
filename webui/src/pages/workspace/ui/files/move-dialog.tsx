// Purpose: Choose a server-authorized destination for a file move without drag gestures.
// Pattern: Astryx folder picker over revision-checked native mutations.
// Usage: EntryDialog mounts this only for a selected move action.
// Related: webui/src/pages/workspace/ui/files/entry-dialog.tsx, webui/src/pages/workspace/model/file-mutations.ts.
import { Banner } from '@astryxdesign/core/Banner';
import { Button } from '@astryxdesign/core/Button';
import { Dialog, DialogHeader } from '@astryxdesign/core/Dialog';
import { HStack, VStack } from '@astryxdesign/core/Layout';
import { List, ListItem } from '@astryxdesign/core/List';
import { ScrollableArea } from '@astryxdesign/core/ScrollableArea';
import { Text } from '@astryxdesign/core/Text';
import { TextInput } from '@astryxdesign/core/TextInput';
import { useQuery } from '@tanstack/react-query';
import { useState } from 'react';
import { useWorkspaceStore } from '@/pages/workspace/model/store';
import type { DirectoryEntry } from '@/pages/workspace/ui/files/entry-row';
import { call, type FileMutationJson } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { describeError } from '@/shared/lib/errors';

export function MoveDialog({
  entry,
  close,
  submit,
}: {
  entry: DirectoryEntry;
  close: () => void;
  submit: (change: FileMutationJson) => void;
}) {
  const t = useLocale(),
    workspace = useWorkspaceStore((s) => s.selection.workspace);
  const [path, setPath] = useState('.'),
    [input, setInput] = useState('.'),
    [offset, setOffset] = useState(0),
    [revision, setRevision] = useState('');
  const folder = useQuery({
    queryKey: ['move-folder', workspace, path, offset, offset ? revision : ''],
    gcTime: 0,
    queryFn: async ({ signal }) =>
      (
        await call(
          workspace,
          { directory: { path, offset, limit: 200, revision: offset ? revision : undefined } },
          signal
        )
      ).directory!,
  });
  const navigate = (next: string) => {
    setPath(next);
    setInput(next);
    setOffset(0);
    setRevision('');
  };
  const current = folder.data?.path ?? path;
  return (
    <Dialog
      isOpen
      width={640}
      purpose="form"
      onOpenChange={(open) => {
        if (!open) close();
      }}
    >
      <DialogHeader title={t('moveEntry')} subtitle={entry.name} onOpenChange={close} />
      <VStack gap={4} padding={4}>
        <VStack
          as="form"
          gap={3}
          onSubmit={(event) => {
            event.preventDefault();
            if (input.trim()) navigate(input.trim());
          }}
        >
          <TextInput
            label={t('moveDestination')}
            value={input}
            onChange={setInput}
            description={t('moveHelp')}
          />
          <Button type="submit" label={t('openFolder')} />
        </VStack>
        {folder.error && (
          <>
            <Banner status="error" title={describeError(folder.error)} />
            <Button
              label={t('retry')}
              onClick={() => {
                if (offset) navigate(path);
                else void folder.refetch();
              }}
            />
          </>
        )}
        <Button
          label={t('parent')}
          isDisabled={current === '.' || current === '/'}
          onClick={() => {
            const slash = current.lastIndexOf('/');
            navigate(slash < 0 ? '.' : current.slice(0, slash) || '/');
          }}
        />
        <ScrollableArea label={t('moveDestination')} height="min(35dvh, 20rem)">
          <List>
            {folder.data?.items
              ?.filter((item) => item.directory)
              .map((item) => {
                const next = `${current.replace(/\/$/, '')}/${item.name}`;
                return (
                  <ListItem
                    key={item.name}
                    label={item.name ?? ''}
                    onClick={() => navigate(next)}
                  />
                );
              })}
          </List>
        </ScrollableArea>
        <HStack gap={2} justify="end">
          <Button
            label={t('pagePrevious')}
            isDisabled={offset === 0 || folder.isFetching}
            onClick={() => setOffset(Math.max(0, offset - 200))}
          />
          <Button
            label={t('pageNext')}
            isDisabled={!folder.data?.more || folder.isFetching}
            onClick={() => {
              setRevision(folder.data?.revision ?? '');
              setOffset(folder.data?.nextOffset ?? offset);
            }}
          />
        </HStack>
        <Text className="break-all">{current}</Text>
        <HStack gap={2} justify="end">
          <Button label={t('cancel')} onClick={close} />
          <Button
            label={t('moveEntry')}
            variant="primary"
            isDisabled={
              !folder.isSuccess ||
              folder.isFetching ||
              current === entry.path ||
              current.startsWith(`${entry.path}/`)
            }
            onClick={() => submit({ path: entry.path, revision: entry.revision, folder: current })}
          />
        </HStack>
      </VStack>
    </Dialog>
  );
}
