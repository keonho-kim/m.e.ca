// Purpose: Offer image attachment and skill selection from the composer's native action menu.
// Pattern: Astryx menu with a persistent file-input owner.
// Usage: Composer keeps the image picker mounted while menu actions close.
// Related: webui/src/pages/workspace/ui/composer/composer.tsx, webui/src/pages/workspace/ui/skills/skill-picker.tsx.
import { DropdownMenu, DropdownMenuItem } from '@astryxdesign/core/DropdownMenu';
import { FileInput } from '@astryxdesign/core/FileInput';
import { Icon } from '@astryxdesign/core/Icon';
import { IconButton } from '@astryxdesign/core/IconButton';
import { VStack } from '@astryxdesign/core/Layout';
import { useRef } from 'react';
import { useLocale } from '@/shared/i18n';

export function AttachmentMenu({
  disabled,
  attach,
  openSkills,
  maxFiles,
  maxSize,
}: {
  disabled: boolean;
  attach: (files: File[]) => void;
  openSkills: () => void;
  maxFiles: number;
  maxSize: number;
}) {
  const t = useLocale();
  const input = useRef<HTMLInputElement>(null);
  return (
    <>
      <DropdownMenu
        placement="above"
        alignment="start"
        menuWidth={256}
        renderTrigger={(props) => (
          <IconButton
            {...props}
            label={t('composerMenu')}
            variant="attachment"
            icon={<Icon icon="meca:add" />}
            isDisabled={disabled}
          />
        )}
      >
        <DropdownMenuItem
          label={t('attachment')}
          icon="meca:image"
          onClick={() => input.current?.click()}
        />
        <DropdownMenuItem label={t('browseSkills')} icon="meca:book" onClick={openSkills} />
      </DropdownMenu>
      <VStack hidden>
        <FileInput
          ref={input}
          label={t('attachment')}
          value={null}
          accept="image/*"
          isMultiple
          maxFiles={maxFiles}
          maxSize={maxSize}
          onChange={(files) => {
            if (files) attach(Array.isArray(files) ? files : [files]);
          }}
        />
      </VStack>
    </>
  );
}
