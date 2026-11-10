// Purpose: Resolve file extensions to the official Astryx syntax language identifiers.
// Pattern: Deterministic language selection.
// Usage: The CodeEditor and code preview share this mapping.
// Related: webui/src/shared/ui/code-editor/index.tsx, webui/src/pages/workspace/ui/files/file-document.tsx.
export function codeLanguage(path: string) {
  const ext = path.split('.').at(-1)?.toLowerCase() ?? '';
  return (
    (
      {
        js: 'javascript',
        mjs: 'javascript',
        ts: 'typescript',
        py: 'python',
        sh: 'bash',
        yml: 'yaml',
        md: 'markdown',
      } as Record<string, string>
    )[ext] ?? ext
  );
}
