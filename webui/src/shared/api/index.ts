// Purpose: Expose the native HTTP transport and generated JSON contracts.
// Pattern: FSD shared API public entry.
// Usage: Features, entities and workspace code import transport contracts here.
// Related: webui/src/shared/api/client.ts, api/meca/v1.
export { call, control, imageURL, loadThumbnail, releaseUpload, upload } from '@/shared/api/client';
export { subscribeEventStream } from '@/shared/api/event-stream';
export type {
  EditorDocumentJson,
  FileMutationJson,
  FileMutationResultJson,
  WorkspaceJson,
} from '@/shared/api/gen/meca/v1/application_pb';
export type { CommandJson, DetailRequestJson } from '@/shared/api/gen/meca/v1/control_pb';
export type { EventJson, StatisticsJson } from '@/shared/api/gen/meca/v1/events_pb';
export type {
  HistoryItemJson,
  NotificationJson,
  ReplyJson,
  RunCompletionJson,
  SessionJson,
  SnapshotJson,
} from '@/shared/api/gen/meca/v1/state_pb';
export type {
  CapabilitiesJson,
  ModelCandidateJson,
  ProviderJson,
  TaskJson,
} from '@/shared/api/gen/meca/v1/workspace_pb';
export { nativeJSON, nativeRequest } from '@/shared/api/http';
export { queries } from '@/shared/api/query-client';
