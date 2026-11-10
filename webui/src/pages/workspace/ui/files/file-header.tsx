// Purpose: Present the file title, save actions and Preview/Raw selection without clipping controls.
// Pattern: Astryx document header with controlled presentation actions.
// Usage: FileDocument owns drafts and persistence; this header only reports user choices.
// Related: webui/src/pages/workspace/ui/files/file-document.tsx.
import { Button } from '@astryxdesign/core/Button';
import { Icon } from '@astryxdesign/core/Icon';
import { IconButton } from '@astryxdesign/core/IconButton';
import { HStack } from '@astryxdesign/core/Layout';
import { SegmentedControl, SegmentedControlItem } from '@astryxdesign/core/SegmentedControl';
import { Text } from '@astryxdesign/core/Text';
import { useLocale } from '@/shared/i18n';

export function FileHeader({
  path,
  editing,
  dirty,
  busy,
  ready,
  readonly,
  markdown,
  mode,
  changeMode,
  save,
  cancel,
  showFiles,
}: {
  path: string;
  editing: boolean;
  dirty: boolean;
  busy: boolean;
  ready: boolean;
  readonly: boolean;
  markdown: boolean;
  mode: 'preview' | 'raw';
  changeMode: (mode: string) => void;
  save: () => void;
  cancel: () => void;
  showFiles?: () => void;
}) {
  const t = useLocale();
  return (
    <HStack
      className="document-toolbar min-w-0"
      role="group"
      aria-label={path}
      padding={3}
      gap={3}
      justify="between"
      wrap="wrap"
    >
      <HStack gap={2} className="min-w-0 grow basis-48">
        {showFiles && (
          <IconButton
            label={t('backToFiles')}
            variant="ghost"
            icon={<Icon icon={'meca:folder'} />}
            onClick={showFiles}
          />
        )}
        <Text type="label" maxLines={1} hasTruncateTooltip className="min-w-0 grow">
          {path}
        </Text>
      </HStack>
      <HStack gap={2} wrap="wrap" justify="end" className="min-w-0 ms-auto">
        {editing ? (
          <>
            <Button label={t('cancel')} isDisabled={busy || !dirty} onClick={cancel} />
            <Button
              label={t('save')}
              variant="primary"
              isLoading={busy}
              isDisabled={!dirty || !ready}
              onClick={save}
            />
          </>
        ) : readonly ? (
          <Text type="supporting" color="secondary">
            {t('readonly')}
          </Text>
        ) : null}
        <SegmentedControl
          label={t('fileView')}
          value={mode}
          onChange={changeMode}
          size="sm"
          isDisabled={busy || (editing && !ready)}
        >
          <SegmentedControlItem value="preview" label={t('preview')} isDisabled={!markdown} />
          <SegmentedControlItem value="raw" label={t('raw')} />
        </SegmentedControl>
      </HStack>
    </HStack>
  );
}
