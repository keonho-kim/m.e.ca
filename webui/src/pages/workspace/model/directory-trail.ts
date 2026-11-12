// Purpose: Parse navigable POSIX directory ancestors without conflating labels with server paths.
// Pattern: Pure file-navigation projection.
// Usage: DirectoryBreadcrumbs uses these values for visible links and collapsed ancestors.
// Related: webui/src/pages/workspace/ui/files/directory-breadcrumbs.tsx.
export function directoryTrail(path: string) {
  const result: { label: string; path: string }[] = [];
  const parts = path.split('/');
  let prefix = '';
  const absolute = path.startsWith('/');
  for (const label of parts) {
    if (!label || label === '.') continue;
    prefix = prefix ? `${prefix}/${label}` : `${absolute ? '/' : ''}${label}`;
    result.push({ label, path: prefix });
  }
  return result;
}
