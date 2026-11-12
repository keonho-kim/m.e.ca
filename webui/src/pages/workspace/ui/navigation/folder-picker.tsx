// Purpose: Browse, create and explicitly select server-side workspace folders.
// Pattern: Astryx folder chooser with separate selection and navigation.
// Usage: Sidebar opens this dialog; the native API owns filesystem changes.
// Related: internal/serverapi/application.go, internal/fileview/directory_create.go.
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
import { registerWorkspace } from '@/pages/workspace/model/workspace-picker';
import { call } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { describeError, failure } from '@/shared/lib/errors';

export default function FolderPicker({ navigate }: { navigate: (target: Target) => void }) {
  const t = useLocale();
  const [path, setPath] = useState(''),
    [entry, setEntry] = useState('');
  const [selected, setSelected] = useState('');
  const [busy, setBusy] = useState(false),
    [error, setError] = useState('');
  const [creating, setCreating] = useState(false),
    [name, setName] = useState('');
  const [offset, setOffset] = useState(0);
  const changePath = (value: string) => {
    setPath(value);
    setOffset(0);
    setEntry('');
    setSelected('');
    setError('');
    setCreating(false);
  };
  const folder = useQuery({
    queryKey: ['folder-picker', path, offset],
    gcTime: 0,
    queryFn: async ({ signal }) => {
      const reply = await call('', { browseFolders: { path, offset } }, signal);
      if (!reply.directory?.path) throw failure('invalid_response');
      return reply.directory;
    },
  });
  const choose = async (value = selected || folder.data?.path) => {
    if (!value || busy) return;
    setBusy(true);
    setError('');
    try {
      navigate(await registerWorkspace(value));
      useWorkspaceStore.getState().set({ picker: false });
    } catch (error) {
      setError(describeError(error));
      reportError(error);
    } finally {
      setBusy(false);
    }
  };
  const create = async () => {
    if (!folder.data?.path || !name.trim() || busy) return;
    setBusy(true);
    setError('');
    try {
      const reply = await call('', { createFolder: { path: folder.data.path, name } });
      if (!reply.directory?.path) throw failure('invalid_response');
      setSelected(reply.directory.path);
      setCreating(false);
      setName('');
      await folder.refetch();
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
      width={960}
      className="folder-modal"
    >
      <Layout
        header={<DialogHeader title={t('addWorkspace')} onOpenChange={close} />}
        content={
          <LayoutContent padding={5}>
            <VStack gap={4}>
              {error && <Banner status="error" title={error} />}
              <HStack
                as="form"
                gap={3}
                onSubmit={(event) => {
                  event.preventDefault();
                  if (entry.trim()) changePath(entry.trim());
                }}
              >
                <VStack className="flex-1 min-w-0">
                  <TextInput
                    label={t('path')}
                    width="100%"
                    value={entry}
                    placeholder={folder.data?.path ?? '~'}
                    onChange={setEntry}
                    isDisabled={busy}
                  />
                </VStack>
                <Button
                  className="shrink-0"
                  type="button"
                  label={t('newFolder')}
                  icon={<Icon icon="meca:folder-add" />}
                  isDisabled={busy || !folder.data}
                  onClick={() => setCreating(true)}
                />
              </HStack>
              {creating && (
                <HStack
                  as="form"
                  gap={3}
                  onSubmit={(event) => {
                    event.preventDefault();
                    void create();
                  }}
                >
                  <VStack className="flex-1 min-w-0">
                    <TextInput
                      label={t('folderName')}
                      value={name}
                      onChange={setName}
                      hasAutoFocus
                      isDisabled={busy}
                    />
                  </VStack>
                  <Button
                    className="shrink-0"
                    type="submit"
                    label={t('create')}
                    isLoading={busy}
                    isDisabled={!name.trim()}
                  />
                  <Button
                    className="shrink-0"
                    label={t('cancel')}
                    isDisabled={busy}
                    onClick={() => setCreating(false)}
                  />
                </HStack>
              )}
              <HStack gap={2}>
                <IconButton
                  label={t('parent')}
                  variant="ghost"
                  icon={<Icon icon="arrowUp" />}
                  isDisabled={busy || !folder.data || folder.data.parent === folder.data.path}
                  onClick={() => changePath(folder.data?.parent ?? '')}
                />
                <Text type="supporting" className="break-all">
                  {folder.data?.path}
                </Text>
              </HStack>
              {folder.isPending && <Spinner label={t('loading')} />}
              {folder.error && <Banner status="error" title={describeError(folder.error)} />}
              {folder.error && <Button label={t('retry')} onClick={() => folder.refetch()} />}
              <List>
                {folder.data?.items?.map((value) => {
                  if (!folder.data.path) return null;
                  const child = `${folder.data.path.replace(/\/$/, '')}/${value.name}`;
                  return (
                    <ListItem
                      key={value.name}
                      label={
                        <Text maxLines={1} hasTruncateTooltip>
                          {value.name}
                        </Text>
                      }
                      startContent={<Icon icon="meca:folder" />}
                      isSelected={selected === child}
                      isDisabled={busy}
                      onClick={() => setSelected(child)}
                      onDoubleClick={() => {
                        void choose(child);
                      }}
                      endContent={
                        <IconButton
                          label={`${value.name} ${t('openFolder')}`}
                          icon={<Icon icon="chevronRight" />}
                          variant="ghost"
                          isDisabled={busy}
                          onClick={(event) => {
                            event.stopPropagation();
                            changePath(child);
                          }}
                        />
                      }
                    />
                  );
                })}
              </List>
              <HStack gap={2} justify="end">
                <Button
                  label={t('back')}
                  isDisabled={!offset || busy}
                  onClick={() => setOffset(Math.max(0, offset - 200))}
                />
                <Button
                  label={t('pageNext')}
                  isDisabled={!folder.data?.more || busy}
                  onClick={() => setOffset(folder.data?.nextOffset ?? offset)}
                />
              </HStack>
            </VStack>
          </LayoutContent>
        }
        footer={
          <VStack padding={4} gap={3}>
            <Text type="supporting" color="secondary" className="break-all">
              {selected || folder.data?.path}
            </Text>
            <HStack gap={3} justify="end">
              <Button
                label={t('selectFolder')}
                variant="primary"
                isLoading={busy}
                isDisabled={!folder.data || !!folder.error}
                onClick={() => choose()}
              />
              <Button label={t('close')} isDisabled={busy} onClick={close} />
            </HStack>
          </VStack>
        }
      />
    </Dialog>
  );
}
