// Purpose: Project native tool lifecycle events into Astryx's supported call states.
// Pattern: Deterministic presentation adapter.
// Usage: Saved history and live activity use the same status meanings.
// Related: webui/src/pages/workspace/ui/chat/history-row.tsx, webui/src/pages/workspace/ui/chat/live-activity.tsx.
import type { ChatToolCallStatus } from '@astryxdesign/core/Chat';
import type { EventJson } from '@/shared/api';

function toolPhase(event: EventJson) {
  return event.tool?.phase ?? 'TOOL_PHASE_UNSPECIFIED';
}
export function terminalTool(event: EventJson) {
  return [
    'TOOL_PHASE_SUCCEEDED',
    'TOOL_PHASE_FAILED',
    'TOOL_PHASE_INTERRUPTED',
    'TOOL_PHASE_SKIPPED',
  ].includes(toolPhase(event));
}
export function toolStatus(event: EventJson): ChatToolCallStatus {
  const phase = toolPhase(event);
  if (phase === 'TOOL_PHASE_RUNNING') return 'running';
  if (
    phase === 'TOOL_PHASE_FAILED' ||
    phase === 'TOOL_PHASE_INTERRUPTED' ||
    phase === 'TOOL_PHASE_SKIPPED'
  )
    return 'error';
  if (phase === 'TOOL_PHASE_SUCCEEDED') return 'complete';
  return 'pending';
}
