// Purpose: Parse bounded tool pages and preserve readable values without rendering protocol envelopes.
// Pattern: Pure presentation projection.
// Usage: ToolValue renders arguments and results; request actions decode bounded public arguments on demand.
// Related: webui/src/pages/workspace/ui/chat/tool-value.tsx, webui/src/pages/workspace/ui/chat/tool-call.tsx.
export function parseToolValue(text: string): unknown {
  try {
    return JSON.parse(text);
  } catch {
    return text;
  }
}
export function toolArguments(encoded?: string): unknown {
  if (!encoded) return undefined;
  try {
    const data = Uint8Array.from(atob(encoded), (char) => char.charCodeAt(0));
    return JSON.parse(new TextDecoder('utf-8', { fatal: true }).decode(data));
  } catch {
    return undefined;
  }
}
export function valueText(value: unknown) {
  return value === undefined
    ? ''
    : typeof value === 'string'
      ? value
      : JSON.stringify(value, null, 2);
}

export function toolFields(value: Record<string, unknown>, request: boolean) {
  const entries = Object.entries(value),
    normalize = (key: string) => key.replaceAll('_', '').toLowerCase();
  if (request) return { content: entries, metadata: [] as [string, unknown][] };
  const body = new Set(['stdout', 'stderr', 'content', 'text', 'output', 'diff', 'patch']);
  const content = entries.filter(
    ([key, data]) => body.has(normalize(key)) && data !== '' && data !== null
  );
  if (!content.length)
    return {
      content: entries.filter(([, data]) => data !== '' && data !== null),
      metadata: [] as [string, unknown][],
    };
  return {
    content,
    metadata: entries.filter(
      ([key, data]) => !body.has(normalize(key)) && data !== '' && data !== null
    ),
  };
}

const labels = {
  command: 'toolCommand',
  path: 'toolPath',
  content: 'toolContent',
  stdout: 'toolStdout',
  stderr: 'toolStderr',
  output: 'toolOutput',
  text: 'toolContent',
  exit_code: 'toolExitCode',
  exitcode: 'toolExitCode',
  status: 'toolStatus',
  directory: 'toolDirectory',
  working_directory: 'toolDirectory',
  pattern: 'toolPattern',
  query: 'toolQuery',
  patch: 'toolChanges',
  diff: 'toolChanges',
} as const;
export function toolFieldName(
  key: string,
  translate: (key: (typeof labels)[keyof typeof labels]) => string
) {
  return Object.hasOwn(labels, key.toLowerCase())
    ? translate(labels[key.toLowerCase() as keyof typeof labels])
    : key.replaceAll('_', ' ');
}
