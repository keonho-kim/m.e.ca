// Purpose: Resolve file extensions to the official Astryx syntax language identifiers.
// Pattern: Deterministic language selection.
// Usage: The simple editor's syntax adapter and Astryx code preview share this mapping.
// Related: webui/src/shared/ui/code-editor/index.tsx, webui/src/pages/workspace/ui/files/file-preview.tsx.
import { fileType, fileTypes } from '@/shared/lib/code-language/file-types';

export { fileType, fileTypes } from '@/shared/lib/code-language/file-types';
export function codeLanguage(path: string) {
  const name = path.slice(path.lastIndexOf('/') + 1).toLowerCase();
  if (name === 'angular.json') return 'json';
  if (/^(docker-)?compose\.ya?ml$/.test(name)) return 'yaml';
  const type = fileType(path);
  if (type) return /\.jsx$/i.test(path) ? 'jsx' : fileTypes[type][1];
  const ext = path.slice(path.lastIndexOf('.') + 1).toLowerCase();
  return ext;
}
