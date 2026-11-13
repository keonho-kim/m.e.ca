// Purpose: Parse native reply envelopes before a response can cross a workspace or operation boundary.
// Pattern: Allocation-light transport parser over generated JSON contracts.
// Usage: The HTTP command client checks exactly one expected result without copying large bodies.
// Related: api/meca/v1/control.proto, api/meca/v1/state.proto, webui/src/shared/api/client.ts.
import type { CommandJson, ReplyJson } from '@/shared/api';
import { parseHistory, parseSnapshot } from '@/shared/api/transcript';
import { boolean, integer, text } from '@/shared/api/transcript-values';
import { ApiError, failure } from '@/shared/lib/errors';

const results = {
  listSessions: 'sessions',
  selectSession: 'state',
  closeSession: 'ok',
  state: 'state',
  history: 'history',
  searchHistory: 'historyMatches',
  detail: 'detail',
  setModel: 'state',
  setSettings: 'state',
  start: 'task',
  taskStatus: 'task',
  cancel: 'ok',
  approve: 'ok',
  steer: 'ok',
  listSkills: 'skills',
  getSkill: 'skill',
  readSkillFile: 'skillFile',
  capabilities: 'capabilities',
  providers: 'providers',
  discoverModels: 'models',
  directory: 'directory',
  previewFile: 'filePreview',
  fileSelections: 'fileSelections',
  selectFile: 'ok',
  historyImage: 'image',
  draft: 'draft',
  listTasks: 'tasks',
  commands: 'commands',
  resources: 'resources',
  listWorkspaces: 'workspaces',
  addWorkspace: 'workspaces',
  deleteSession: 'ok',
  commandCatalog: 'commandCatalog',
  parseInput: 'parsedInput',
  editorRead: 'editor',
  editorSave: 'editor',
  diagnoseModel: 'diagnostic',
  browseFolders: 'directory',
  lookupRequest: 'task',
  suggestSkills: 'skills',
  createFolder: 'directory',
  mutateFile: 'fileMutation',
  sessionUsage: 'usage',
  editorProbe: 'editorVersion',
  lookupMutation: 'fileMutation',
} as const satisfies Record<Exclude<keyof CommandJson, 'id' | 'workspaceId'>, keyof ReplyJson>;
export function commandResult(command: CommandJson) {
  const operations = Object.keys(command).filter((key) => key !== 'id' && key !== 'workspaceId');
  const [operation] = operations;
  if (operations.length !== 1 || !operation || !Object.hasOwn(results, operation))
    throw failure('invalid_response');
  return results[operation as keyof typeof results];
}
function object(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object' && !Array.isArray(value);
}
function nativeError(payload: Record<string, unknown>) {
  if (typeof payload.code !== 'string' || !payload.code || typeof payload.message !== 'string')
    return failure('invalid_response');
  return new ApiError(payload.code, payload.message, true);
}
export function replyFailure(body: unknown) {
  return object(body) && object(body.error) ? nativeError(body.error) : failure('network');
}
export function parseReply(
  body: unknown,
  expected: keyof ReplyJson,
  workspace?: string,
  id?: string
): ReplyJson {
  if (!object(body)) throw failure('invalid_response');
  if (
    (id !== undefined && body.id !== undefined && body.id !== id) ||
    (workspace !== undefined && body.workspaceId !== undefined && body.workspaceId !== workspace)
  )
    throw failure('invalid_response');
  const payloads = Object.keys(body).filter((key) => key !== 'id' && key !== 'workspaceId');
  const [key] = payloads;
  if (payloads.length !== 1 || !key) throw failure('invalid_response');
  const payload = body[key];
  if (!object(payload)) throw failure('invalid_response');
  if (key === 'error') {
    throw nativeError(payload);
  }
  if (key !== expected) throw failure('invalid_response');
  if (key === 'history') parseHistory(payload);
  if (key === 'state') parseSnapshot(payload);
  if (key === 'historyMatches') {
    text(payload.cursor, 1024);
    if (Object.keys(payload).some((key) => key !== 'items' && key !== 'cursor'))
      throw failure('invalid_response');
    if (payload.items !== undefined) {
      if (!Array.isArray(payload.items) || payload.items.length > 32)
        throw failure('invalid_response');
      for (const item of payload.items) {
        if (!object(item)) throw failure('invalid_response');
        integer(item.sequence);
        integer(item.offset);
        boolean(item.thinking);
        text(item.excerpt, 2048);
        text(item.revision, 64);
      }
    }
  }
  return body;
}
