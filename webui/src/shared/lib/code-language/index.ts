// Purpose: Resolve file extensions to the official Astryx syntax language identifiers.
// Pattern: Deterministic language selection.
// Usage: The simple editor's syntax adapter and Astryx code preview share this mapping.
// Related: webui/src/shared/ui/code-editor/index.tsx, webui/src/pages/workspace/ui/files/file-preview.tsx.
const languages: Readonly<Record<string, string>> = {
  js: 'javascript',
  mjs: 'javascript',
  ts: 'typescript',
  py: 'python',
  sh: 'bash',
  yml: 'yaml',
  md: 'markdown',
};
export function codeLanguage(path: string) {
  const ext = path.slice(path.lastIndexOf('.') + 1).toLowerCase();
  return languages[ext] ?? ext;
}
