// Purpose: Render build-compiled Material glyphs through the Astryx icon contract.
// Pattern: File-type presentation adapter with no MUI styling runtime.
// Usage: File rows and the document header show these icons after the Files surface loads.
// Related: webui/src/shared/lib/code-language/file-types.ts, webui/src/pages/workspace/ui/files/entry-row.tsx.
import { Icon } from '@astryxdesign/core/Icon';
import { fileType, fileTypes } from '@/shared/lib/code-language';
import { fileGlyphs } from '@/shared/ui/file-icon/generated';
export function FileIcon({ path }: { path: string }) {
  const type = fileType(path);
  return (
    <Icon
      icon={fileGlyphs[type ? fileTypes[type][2] : 'file']}
      data-file-type={type ?? 'file'}
      className="shrink-0"
    />
  );
}
