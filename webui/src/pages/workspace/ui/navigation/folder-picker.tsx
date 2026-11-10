// Purpose: Select and register a server-side project directory without reading file contents.
// Pattern: Explicit folder-selection flow.
// Usage: Sidebar folder addition opens this modal.
// Related: webui/src/pages/workspace/ui/navigation/sidebar.tsx, internal/serverapi/application.go.

import { Banner } from '@astryxdesign/core/Banner';
import { Button } from '@astryxdesign/core/Button';
import { Dialog, DialogHeader } from '@astryxdesign/core/Dialog';
import { Icon } from '@astryxdesign/core/Icon';
import { IconButton } from '@astryxdesign/core/IconButton';
import { HStack, Layout, LayoutContent, VStack } from '@astryxdesign/core/Layout';
import { List, ListItem } from '@astryxdesign/core/List';
import { Spinner } from '@astryxdesign/core/Spinner';
import { Text } from '@astryxdesign/core/Text';
import { TextInput } from '@astryxdesign/core/TextInput';
import { useQuery } from '@tanstack/react-query';
import { useState } from 'react';
import { reportError } from '@/entities/server-session';
import { type Target, useWorkspaceStore } from '@/pages/workspace/model/store';
import { call, queries } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { describeError } from '@/shared/lib/errors';
export default function FolderPicker({ navigate }: { navigate: (target: Target) => void }) {
  const t = useLocale(),
    [path, setPath] = useState(''),
    [entry, setEntry] = useState(''),
    [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [offset, setOffset] = useState(0);
  const changePath = (value: string) => {
    setPath(value);
    setOffset(0);
    setEntry('');
  };
  const folder = useQuery({
    queryKey: ['folder-picker', path, offset],
    queryFn: async ({ signal }) =>
      (await call('', { browseFolders: { path, offset } }, signal)).directory!,
    gcTime: 0,
  });
  const choose = async () => {
    if (!folder.data?.path) return;
    setBusy(true);
    try {
      const value = folder.data.path;
      const reply = await call('', { addWorkspace: { path: value } });
      queries.setQueryData(['workspaces'], reply.workspaces);
      const workspace = reply.workspaces?.items?.find((v) => v.path === value);
      if (workspace?.id) navigate({ workspace: workspace.id, session: '' });
      useWorkspaceStore.getState().set({ picker: false });
    } catch (error) {
      setError(describeError(error));
      reportError(error);
    } finally {
      setBusy(false);
    }
  };
  const close = () => {
    if (!busy) useWorkspaceStore.getState().set({ picker: false });
  };
  return (
    <Dialog
      isOpen
      onOpenChange={(open) => {
        if (!open) close();
      }}
      purpose="form"
      width={640}
      className="folder-modal"
    >
      <Layout
        header={<DialogHeader title={t('addWorkspace')} onOpenChange={close} />}
        content={
          <LayoutContent padding={4}>
            <VStack gap={4}>
              {error && <Banner status="error" title={error} />}
              <HStack
                as="form"
                gap={2}
                onSubmit={(event) => {
                  event.preventDefault();
                  changePath(entry);
                }}
              >
                <TextInput
                  label={t('path')}
                  width="100%"
                  value={entry}
                  placeholder={folder.data?.path ?? '~'}
                  onChange={setEntry}
                />
                <Button type="submit" label={t('open')} />
              </HStack>
              <HStack gap={2} className="folder-current">
                <IconButton
                  label={t('parent')}
                  variant="ghost"
                  icon={<Icon icon={'arrowUp'} />}
                  onClick={() => changePath(folder.data?.parent ?? '')}
                />
                <Text type="supporting" maxLines={1} hasTruncateTooltip>
                  {folder.data?.path}
                </Text>
              </HStack>
              {folder.isPending && <Spinner label={t('loading')} />}
              {folder.error && <Button label={t('retry')} onClick={() => folder.refetch()} />}
              <List>
                {folder.data?.items?.map((value) => (
                  <ListItem
                    key={value.name}
                    label={value.name}
                    startContent={<Icon icon={'meca:folder'} />}
                    onClick={() => changePath(`${folder.data.path}/${value.name}`)}
                  />
                ))}
              </List>
              <HStack gap={2} justify="end">
                <Button
                  label={t('back')}
                  isDisabled={!offset}
                  onClick={() => setOffset(Math.max(0, offset - 200))}
                />
                <Button
                  label={t('pageNext')}
                  isDisabled={!folder.data?.more}
                  onClick={() => setOffset(folder.data?.nextOffset ?? offset)}
                />
              </HStack>
            </VStack>
          </LayoutContent>
        }
        footer={
          <HStack gap={2} justify="end" padding={4}>
            <Button label={t('cancel')} isDisabled={busy} onClick={close} />
            <Button
              label={t('selectFolder')}
              variant="primary"
              isLoading={busy}
              isDisabled={!folder.data}
              onClick={choose}
            />
          </HStack>
        }
      />
    </Dialog>
  );
}
