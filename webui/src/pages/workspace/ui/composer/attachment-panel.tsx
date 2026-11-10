// Purpose: Offer skill browsing and image-only attachment inside the official composer drawer.
// Pattern: Astryx drawer frame with composer-owned disclosure and native input budgets.
// Usage: Composer owns expansion; FileInput reports only newly selected images.
// Related: webui/src/pages/workspace/ui/composer/composer.tsx, webui/src/pages/workspace/model/commands.ts.
import { Button } from '@astryxdesign/core/Button';
import { ChatComposerDrawer } from '@astryxdesign/core/Chat';
import { FileInput } from '@astryxdesign/core/FileInput';
import { Icon } from '@astryxdesign/core/Icon';
import { VStack } from '@astryxdesign/core/Layout';
import { view } from '@/pages/workspace/model/commands';
import { useLocale } from '@/shared/i18n';
export function AttachmentPanel({
  id,
  expanded,
  onExpandedChange,
  attach,
  maxFiles,
  maxSize,
}: {
  id: string;
  expanded: boolean;
  onExpandedChange: (value: boolean) => void;
  attach: (files: File[]) => void;
  maxFiles: number;
  maxSize: number;
}) {
  const t = useLocale();
  return (
    <ChatComposerDrawer
      id={id}
      label={t('composerMenu')}
      data-menu-open={expanded ? 'true' : 'false'}
      aria-hidden={!expanded}
      inert={!expanded}
    >
      <VStack width="100%" gap={3} paddingBlockEnd={2}>
        <Button
          label={t('browseSkills')}
          variant="ghost"
          icon={<Icon icon={'meca:book'} />}
          onClick={() => {
            if (view('skills')) onExpandedChange(false);
          }}
        />
        <FileInput
          label={t('attachment')}
          value={null}
          accept="image/*"
          isMultiple
          maxSize={maxSize}
          maxFiles={maxFiles}
          onChange={(files) => {
            if (files) attach(Array.isArray(files) ? files : [files]);
            onExpandedChange(false);
          }}
        />
      </VStack>
    </ChatComposerDrawer>
  );
}
