// Purpose: Render one virtualized entry with its reference menu and folder drop target.
// Pattern: Astryx presentation with dnd-kit pointer and keyboard bindings.
// Usage: DirectoryColumn supplies the visible entry and owns the mutation dialog.
// Related: webui/src/pages/workspace/ui/files/directory-column.tsx, webui/src/pages/workspace/ui/files/entry-dialog.tsx.
import { Button } from '@astryxdesign/core/Button';
import { DropdownMenu, DropdownMenuItem } from '@astryxdesign/core/DropdownMenu';
import { useMergedRefs } from '@astryxdesign/core/hooks';
import { Icon } from '@astryxdesign/core/Icon';
import { IconButton } from '@astryxdesign/core/IconButton';
import { HStack } from '@astryxdesign/core/Layout';
import { Tooltip } from '@astryxdesign/core/Tooltip';
import { useDraggable, useDroppable } from '@dnd-kit/react';
import { useLocale } from '@/shared/i18n';
import { FileIcon } from '@/shared/ui/file-icon';

export type DirectoryEntry = { path: string; name: string; revision: string; directory: boolean };
export type EntryAction = { entry: DirectoryEntry; kind: 'rename' | 'delete' };

export function EntryRow({
  entry,
  top,
  busy,
  referenced,
  canReference,
  open,
  reference,
  selectAction,
}: {
  entry: DirectoryEntry;
  top: number;
  busy: boolean;
  referenced: boolean;
  canReference: boolean;
  open: (path: string, directory: boolean) => void;
  reference: (path: string, selected: boolean) => void;
  selectAction: (action: EntryAction) => void;
}) {
  const t = useLocale();
  const { path, name, directory, revision } = entry;
  const drag = useDraggable({ id: path, data: { path, revision }, disabled: busy || !revision });
  const drop = useDroppable({
    id: `folder:${path}`,
    data: { folder: path },
    disabled: busy || !directory,
  });
  const ref = useMergedRefs(drag.ref, drop.ref);
  return (
    <HStack
      ref={ref}
      className="directory-row"
      style={{ position: 'absolute', width: '100%', top }}
      align="center"
      gap={1}
    >
      <Tooltip content={name}>
        <Button
          ref={drag.handleRef}
          aria-keyshortcuts="Alt+Space"
          label={name}
          className="grow min-w-0 justify-start cursor-grab"
          variant={drop.isDropTarget ? 'primary' : 'ghost'}
          icon={directory ? <Icon icon="meca:folder" /> : <FileIcon path={path} />}
          endContent={
            directory ? (
              <HStack align="center" justify="center" className="self-center shrink-0">
                <Icon icon="chevronRight" size="sm" />
              </HStack>
            ) : undefined
          }
          isDisabled={busy}
          onClick={() => open(path, directory)}
        />
      </Tooltip>
      <DropdownMenu
        alignment="end"
        renderTrigger={(props) => (
          <IconButton
            {...props}
            label={`${t('entryActions')} · ${name}`}
            variant="ghost"
            size="sm"
            icon={<Icon icon="meca:file-actions" />}
            isDisabled={busy}
          />
        )}
      >
        <DropdownMenuItem
          label={t(referenced ? 'removeReference' : 'addReference')}
          icon="meca:add"
          isDisabled={!canReference}
          onClick={() => reference(directory ? `${path}/` : path, !referenced)}
        />
        <DropdownMenuItem
          label={t('renameEntry')}
          icon="meca:edit"
          isDisabled={!revision}
          onClick={() => selectAction({ entry, kind: 'rename' })}
        />
        <DropdownMenuItem
          label={t('deleteEntry')}
          icon="meca:delete"
          variant="destructive"
          isDisabled={!revision}
          onClick={() => selectAction({ entry, kind: 'delete' })}
        />
      </DropdownMenu>
    </HStack>
  );
}

export function ParentFolderButton({
  path,
  open,
  busy,
}: {
  path: string;
  open: () => void;
  busy: boolean;
}) {
  const t = useLocale();
  const drop = useDroppable({ id: `folder:${path}`, data: { folder: path }, disabled: busy });
  return (
    <IconButton
      ref={drop.ref}
      label={t('parent')}
      variant={drop.isDropTarget ? 'primary' : 'ghost'}
      isDisabled={busy}
      icon={<Icon icon="meca:back" />}
      onClick={open}
    />
  );
}
