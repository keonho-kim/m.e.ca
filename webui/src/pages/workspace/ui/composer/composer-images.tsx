// Purpose: Present draft images as one native island and release every preview URL on removal.
// Pattern: Memoized attachment view with component-owned browser resources.
// Usage: Composer renders this above its text surface; originals remain only in transient input state.
// Related: webui/src/pages/workspace/model/store.ts, webui/src/pages/workspace/ui/composer/composer.tsx.
import { Carousel } from '@astryxdesign/core/Carousel';
import { useContainerReveal } from '@astryxdesign/core/hooks';
import { Icon } from '@astryxdesign/core/Icon';
import { IconButton } from '@astryxdesign/core/IconButton';
import { VStack } from '@astryxdesign/core/Layout';
import { Thumbnail } from '@astryxdesign/core/Thumbnail';
import { mergeProps } from '@astryxdesign/core/utils';
import { memo, useCallback, useEffect, useState } from 'react';
import { type Target, targetKey, useWorkspaceStore } from '@/pages/workspace/model/store';
import { useLocale } from '@/shared/i18n';

const keys = new WeakMap<File, string>();
function fileKey(file: File) {
  let key = keys.get(file);
  if (!key) {
    key = crypto.randomUUID();
    keys.set(file, key);
  }
  return key;
}
const ImagePreview = memo(function ImagePreview({
  file,
  occurrence,
  remove,
}: {
  file: File;
  occurrence: number;
  remove: (file: File, occurrence: number) => void;
}) {
  const t = useLocale();
  const { getContainerProps, getContentRevealProps } = useContainerReveal();
  const [src, setSrc] = useState('');
  useEffect(() => {
    const url = URL.createObjectURL(file);
    setSrc(url);
    return () => URL.revokeObjectURL(url);
  }, [file]);
  return (
    <VStack
      {...mergeProps(getContainerProps(), { className: 'composer-image relative' })}
      width="max-content"
      gap={0}
    >
      <Thumbnail
        src={src || undefined}
        label={file.name}
        alt={t('imagePreview')}
        isLoading={!src}
      />
      <IconButton
        {...getContentRevealProps({ isLayoutPreserved: true })}
        label={`${file.name} ${t('remove')}`}
        variant="image-remove"
        size="sm"
        icon={<Icon icon="close" size="xsm" />}
        onClick={() => remove(file, occurrence)}
      />
    </VStack>
  );
});
export const ComposerImages = memo(function ComposerImages({
  target,
  onEmpty,
}: {
  target: Target;
  onEmpty: () => void;
}) {
  const t = useLocale();
  const files = useWorkspaceStore((state) => state.inputs[targetKey(target)]?.files);
  const remove = useCallback(
    (file: File, occurrence: number) => {
      const state = useWorkspaceStore.getState();
      const current = state.inputs[targetKey(target)];
      if (!current) return;
      let seen = 0;
      const index = current.files.findIndex((value) => value === file && seen++ === occurrence);
      if (index < 0) return;
      const next = [...current.files.slice(0, index), ...current.files.slice(index + 1)];
      if (state.patchInput(target, { files: next }) && !next.length) onEmpty();
    },
    [target, onEmpty]
  );
  if (!files?.length) return null;
  const occurrences = new Map<File, number>();
  return (
    <Carousel
      aria-label={t('attachedImages')}
      gap={2}
      padding={3}
      hasSnap
      className="composer-images self-start max-w-full"
    >
      {files.map((file) => {
        const occurrence = occurrences.get(file) ?? 0;
        occurrences.set(file, occurrence + 1);
        return (
          <ImagePreview
            key={`${fileKey(file)}:${occurrence}`}
            file={file}
            occurrence={occurrence}
            remove={remove}
          />
        );
      })}
    </Carousel>
  );
});
