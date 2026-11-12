// Purpose: Carry Protobuf content exclusivity into browser application types.
// Pattern: Consumer-owned discriminated transport values.
// Usage: All browser consumers use these contracts after runtime parsing.
// Related: webui/src/shared/api/transcript.ts, api/meca/v1/state.proto.
import type {
  NoticeJson,
  RunCompletionJson,
  TeamContributionJson,
  ToolJson,
  TranscriptMessageJson,
  EventJson as WireEvent,
} from '@/shared/api/gen/meca/v1/events_pb';
import type {
  HistoryItemJson as WireHistory,
  NotificationJson as WireNotification,
  HistoryPageJson as WirePage,
  ReplyJson as WireReply,
  SnapshotJson as WireSnapshot,
} from '@/shared/api/gen/meca/v1/state_pb';

type Exclusive<T> = {
  [K in keyof T]: Required<Pick<T, K>> & Partial<Record<Exclude<keyof T, K>, never>>;
}[keyof T];
type EventBody = {
  message: TranscriptMessageJson;
  work: TeamContributionJson;
  notice: NoticeJson;
  tool: ToolJson;
  completion: RunCompletionJson;
};
export type EventJson = Omit<WireEvent, keyof EventBody> &
  (Exclusive<EventBody> | Partial<Record<keyof EventBody, never>>);
type HistoryBody = {
  message: TranscriptMessageJson;
  event: EventJson;
  completion: RunCompletionJson;
};
export type HistoryItemJson = Omit<WireHistory, keyof HistoryBody> & Exclusive<HistoryBody>;
export type HistoryPageJson = Omit<WirePage, 'items'> & { items?: HistoryItemJson[] };
export type SnapshotJson = Omit<WireSnapshot, 'activity'> & { activity?: EventJson[] };
export type ReplyJson = Omit<WireReply, 'history' | 'state'> & {
  history?: HistoryPageJson;
  state?: SnapshotJson;
};
export type NotificationJson = Omit<WireNotification, 'event' | 'snapshot' | 'reply'> &
  Exclusive<{ event: EventJson; snapshot: SnapshotJson; reply: ReplyJson }>;
