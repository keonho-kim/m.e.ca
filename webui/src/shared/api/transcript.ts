// Purpose: Enforce exclusive transcript bodies at HTTP, live stream and snapshot boundaries.
// Pattern: Validated in-place projection over generated JSON contracts.
// Usage: Reply parsing and the event observer share these runtime checks.
// Related: webui/src/shared/api/transcript-values.ts, webui/src/shared/api/reply.ts.
import type { EventJson, NotificationJson } from '@/shared/api/transcript-contract';
import {
  boolean,
  completion,
  integer,
  message,
  notice,
  object,
  only,
  statistics,
  team,
  text,
} from '@/shared/api/transcript-values';
import { failure } from '@/shared/lib/errors';

const messageKinds = [
  'assistant_delta',
  'thinking',
  'progress',
  'team_progress',
  'team_message',
  'team_discussion',
  'user_message',
];
const toolKinds = [
  'tool_started',
  'tool_waiting',
  'tool_succeeded',
  'tool_failed',
  'tool_interrupted',
  'tool_skipped',
  'command_finished',
  'denied',
  'skill_loaded',
];
const bodies = ['message', 'work', 'notice', 'tool', 'completion'] as const;
function reference(value: unknown) {
  if (value === undefined) return;
  object(value);
  only(value, ['sessionId', 'sequence']);
  text(value.sessionId, 128);
  integer(value.sequence);
  if (!value.sessionId || !value.sequence || value.sequence === '0')
    throw failure('invalid_response');
}
function cursor(value: unknown) {
  if (value === undefined) return;
  if (typeof value !== 'string') throw failure('invalid_response');
  const match = /^([A-Za-z0-9_-]{1,32}):(0|[1-9]\d{0,19})$/.exec(value);
  if (!match) throw failure('invalid_response');
  integer(match[2], 18446744073709551615n);
}
export function parseEvent(value: unknown): EventJson {
  object(value);
  cursor(value.cursor);
  text(value.kind, 128);
  if (!value.kind || ['text', 'thinking', 'team', 'skill'].some((key) => Object.hasOwn(value, key)))
    throw failure('invalid_response');
  const present = bodies.filter((key) => value[key] !== undefined);
  if (present.length > 1) throw failure('invalid_response');
  for (const key of ['historySequence', 'historyThrough', 'sequence', 'generation'])
    integer(
      value[key],
      key === 'sequence' || key === 'generation' ? 18446744073709551615n : undefined
    );
  for (const key of ['agent', 'runId', 'sessionId', 'taskId', 'workspaceId']) text(value[key], 256);
  statistics(value.statistics);
  reference(value.originalReference);
  const body = present[0],
    kind = String(value.kind);
  if (body === 'message') {
    if (!messageKinds.includes(kind)) throw failure('invalid_response');
    message(value.message);
  } else if (body === 'work') {
    if (kind !== 'team_work') throw failure('invalid_response');
    object(value.work);
    only(value.work, ['summary', 'details']);
    if (typeof value.work.summary !== 'string') throw failure('invalid_response');
    text(value.work.summary);
    team(value.work.details);
  } else if (body === 'notice') {
    notice(value.notice);
    const code = (value.notice as Record<string, unknown>).code;
    if (
      code !==
      (
        {
          retry: 'RETRYING',
          compaction_started: 'COMPACTION_STARTED',
          compaction_finished: 'COMPACTION_FINISHED',
          skill_loaded: 'SKILL_READ',
        } as Record<string, string>
      )[kind]
    )
      throw failure('invalid_response');
  } else if (body === 'tool') {
    if (!toolKinds.includes(kind)) throw failure('invalid_response');
    object(value.tool);
    only(value.tool, [
      'phase',
      'executionId',
      'batchId',
      'callId',
      'name',
      'argumentsJson',
      'reference',
      'summary',
      'skill',
    ]);
    for (const key of ['executionId', 'batchId', 'callId', 'name']) text(value.tool[key], 128);
    text(value.tool.summary);
    text(value.tool.argumentsJson);
    reference(value.tool.reference);
    if (value.tool.skill !== undefined) notice({ code: 'SKILL_READ', skill: value.tool.skill });
    if (
      value.tool.phase !== undefined &&
      ![
        'TOOL_PHASE_UNSPECIFIED',
        'TOOL_PHASE_PENDING',
        'TOOL_PHASE_RUNNING',
        'TOOL_PHASE_SUCCEEDED',
        'TOOL_PHASE_FAILED',
        'TOOL_PHASE_INTERRUPTED',
        'TOOL_PHASE_SKIPPED',
      ].includes(String(value.tool.phase))
    )
      throw failure('invalid_response');
  } else if (body === 'completion') {
    if (kind !== 'run_checkpoint') throw failure('invalid_response');
    completion(value.completion);
  } else if (
    messageKinds.includes(kind) ||
    ['team_work', 'retry', 'compaction_started', 'compaction_finished'].includes(kind)
  )
    throw failure('invalid_response');
  return value;
}
export function parseHistory(value: unknown) {
  object(value);
  integer(value.before);
  integer(value.through);
  if (value.items === undefined) return;
  if (!Array.isArray(value.items) || value.items.length > 200) throw failure('invalid_response');
  let through = 0n;
  for (const item of value.items) {
    object(item);
    only(item, ['sequence', 'truncated', 'message', 'event', 'completion']);
    integer(item.sequence);
    boolean(item.truncated);
    if (!item.sequence || item.sequence === '0') throw failure('invalid_response');
    through = BigInt(String(item.sequence)) > through ? BigInt(String(item.sequence)) : through;
    const entries = ['message', 'event', 'completion'].filter((key) => item[key] !== undefined);
    if (entries.length !== 1) throw failure('invalid_response');
    if (item.message !== undefined) message(item.message);
    if (item.event !== undefined) parseEvent(item.event);
    if (item.completion !== undefined) completion(item.completion);
  }
  if (BigInt(String(value.through ?? '0')) < through) throw failure('invalid_response');
}
export function parseSnapshot(value: unknown) {
  object(value);
  cursor(value.cursor);
  if (value.drafts !== undefined) {
    if (!Array.isArray(value.drafts) || value.drafts.length > 128)
      throw failure('invalid_response');
    for (const draft of value.drafts) {
      object(draft);
      text(draft.text);
      text(draft.thinking);
      text(draft.agent, 128);
      text(draft.taskId, 128);
      integer(draft.historySequence);
      integer(draft.generation, 18446744073709551615n);
      boolean(draft.committed);
      boolean(draft.truncated);
    }
  }
  if (value.activity !== undefined) {
    if (!Array.isArray(value.activity) || value.activity.length > 128)
      throw failure('invalid_response');
    for (const event of value.activity) parseEvent(event);
  }
  if (value.task !== undefined) {
    object(value.task);
    integer(value.task.historyThrough);
  }
  if (value.tasks !== undefined) {
    if (!Array.isArray(value.tasks)) throw failure('invalid_response');
    for (const task of value.tasks) {
      object(task);
      integer(task.historyThrough);
    }
  }
}
export function parseNotification(value: unknown): NotificationJson {
  object(value);
  only(value, ['event', 'snapshot']);
  if (Number(value.event !== undefined) + Number(value.snapshot !== undefined) !== 1)
    throw failure('invalid_response');
  if (value.event !== undefined) parseEvent(value.event);
  if (value.snapshot !== undefined) parseSnapshot(value.snapshot);
  return value as NotificationJson;
}
