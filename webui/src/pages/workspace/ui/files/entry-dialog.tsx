// Purpose: Dispatch the selected entry's rename, move or delete confirmation.
// Pattern: One Astryx dialog for the active file action.
// Usage: DirectoryColumn opens the dialog from a virtualized entry menu.
// Related: webui/src/pages/workspace/ui/files/entry-row.tsx, internal/fileview/entry_mutation.go.
import { AlertDialog } from '@astryxdesign/core/AlertDialog';
import { Button } from '@astryxdesign/core/Button';
import { Dialog, DialogHeader } from '@astryxdesign/core/Dialog';
import { HStack, VStack } from '@astryxdesign/core/Layout';
import { TextInput } from '@astryxdesign/core/TextInput';
import { useState } from 'react';
import type { EntryAction } from '@/pages/workspace/ui/files/entry-row';
import { MoveDialog } from '@/pages/workspace/ui/files/move-dialog';
import type { FileMutationJson } from '@/shared/api';
import { useLocale } from '@/shared/i18n';

export function EntryDialog({
  action,
  close,
  mutate,
}: {
  action: EntryAction;
  close: () => void;
  mutate: (change: FileMutationJson) => void;
}) {
  const t = useLocale(),
    [name, setName] = useState(action.entry.name);
  const { path, revision, directory } = action.entry;
  const submit = (change: FileMutationJson) => {
    close();
    mutate(change);
  };
  if (action.kind === 'move')
    return <MoveDialog entry={action.entry} close={close} submit={submit} />;
  if (action.kind === 'delete')
    return (
      <AlertDialog
        isOpen
        title={`${t('deleteEntry')} · ${action.entry.name}`}
        description={t(directory ? 'deleteDirectoryHelp' : 'deleteEntryHelp')}
        onOpenChange={(open) => {
          if (!open) close();
        }}
        actionLabel={t('deleteEntry')}
        cancelLabel={t('cancel')}
        onAction={() => submit({ path, revision, remove: true })}
      />
    );
  const valid =
    name.trim() &&
    name !== action.entry.name &&
    name !== '.' &&
    name !== '..' &&
    !/[/\\\0\r\n]/.test(name);
  return (
    <Dialog
      isOpen
      width={440}
      purpose="form"
      onOpenChange={(open) => {
        if (!open) close();
      }}
    >
      <DialogHeader title={t('renameEntry')} onOpenChange={close} />
      <VStack
        as="form"
        gap={4}
        padding={4}
        onSubmit={(event) => {
          event.preventDefault();
          if (valid) submit({ path, revision, name });
        }}
      >
        <TextInput label={t('entryName')} value={name} onChange={setName} />
        <HStack gap={2} justify="end">
          <Button label={t('cancel')} onClick={close} />
          <Button label={t('save')} type="submit" variant="primary" isDisabled={!valid} />
        </HStack>
      </VStack>
    </Dialog>
  );
}
