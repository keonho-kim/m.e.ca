// Purpose: Expose the native HTTP transport and generated JSON contracts.
// Pattern: FSD shared API public entry.
// Usage: Features, entities and workspace code import transport contracts here.
// Related: webui/src/shared/api/client.ts, api/meca/v1.
export { call, control, imageURL, loadThumbnail, releaseUpload, upload } from '@/shared/api/client';
export type {
  EditorDocumentJson,
  FileMutationJson,
  WorkspaceJson,
} from '@/shared/api/gen/meca/v1/application_pb';
export type { CommandJson, DetailRequestJson } from '@/shared/api/gen/meca/v1/control_pb';
export type { EventJson, StatisticsJson } from '@/shared/api/gen/meca/v1/events_pb';
export type {
  HistoryItemJson,
  NotificationJson,
  ReplyJson,
  SessionJson,
  SnapshotJson,
} from '@/shared/api/gen/meca/v1/state_pb';
export type {
  CapabilitiesJson,
  DraftRequestJson,
  ModelCandidateJson,
  ProviderJson,
  TaskJson,
} from '@/shared/api/gen/meca/v1/workspace_pb';
export { queries } from '@/shared/api/query-client';
