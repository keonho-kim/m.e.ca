// Purpose: Project native tool lifecycle events into Astryx's supported call states.
// Pattern: Deterministic presentation adapter.
// Usage: Saved history and live activity use the same status meanings.
// Related: webui/src/pages/workspace/ui/chat/history-row.tsx, webui/src/pages/workspace/ui/chat/live-activity.tsx.
import type { ChatToolCallStatus } from '@astryxdesign/core/Chat';
export function toolStatus(kind: string | undefined): ChatToolCallStatus {
  if (kind === 'tool_started') return 'running';
  if (kind === 'tool_failed' || kind === 'tool_interrupted' || kind === 'denied') return 'error';
  return 'complete';
}
