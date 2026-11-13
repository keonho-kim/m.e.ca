// Purpose: Extend bounded code-page boundaries without rescanning a stable streaming prefix.
// Pattern: Single-document incremental index with a reset path for source replacement.
// Usage: Each CodePreview owns one index and releases it on unmount.
// Related: webui/src/shared/ui/code-preview/index.tsx.
export function codePages() {
  let previous = '',
    boundaries = [0];
  return (code: string) => {
    // Revisit the last complete boundary: an appended low surrogate may join
    // a previous high surrogate exactly at that boundary.
    if (!code.startsWith(previous)) boundaries = [0];
    else if (boundaries.length > 1) boundaries.pop();
    let start = boundaries.at(-1) ?? 0,
      lines = 0;
    for (let i = start; i < code.length; i++) {
      if (code.charCodeAt(i) === 10) lines++;
      if (lines < 160 && i + 1 - start < 8192) continue;
      if (
        code.charCodeAt(i) >= 0xd800 &&
        code.charCodeAt(i) <= 0xdbff &&
        code.charCodeAt(i + 1) >= 0xdc00 &&
        code.charCodeAt(i + 1) <= 0xdfff
      )
        i++;
      start = i + 1;
      boundaries.push(start);
      lines = 0;
    }
    previous = code;
    return boundaries.at(-1) === code.length && boundaries.length > 1
      ? [...boundaries]
      : [...boundaries, code.length];
  };
}
