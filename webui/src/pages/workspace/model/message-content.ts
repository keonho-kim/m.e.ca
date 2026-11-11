// Purpose: Separate canonical selected-path annotations from user-facing message text.
// Pattern: Pure presentation projection of the engine's reference block.
// Usage: User message bubbles render paths as references instead of showing prompt scaffolding.
// Related: internal/commands/input/request.go, webui/src/pages/workspace/ui/chat/history-row.tsx.
const marker = '\nSelected path references (read only as needed):\n';
export function messageContent(value: string) {
  const index = value.lastIndexOf(marker);
  if (index < 0) return { text: value, paths: [] as string[] };
  const paths = value.slice(index + marker.length).split('\n');
  if (!paths.length || paths.some((path) => !path || path.includes('\r')))
    return { text: value, paths: [] as string[] };
  return { text: value.slice(0, index), paths };
}
