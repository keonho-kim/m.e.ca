// Purpose: Share in-flight snapshots without retaining live response bodies in the query cache.
// Pattern: Reference-counted cancellable network read.
// Usage: The event observer consumes full snapshots; metadata queries discard live bodies.
// Related: webui/src/pages/workspace/api/queries.ts, webui/src/pages/workspace/api/events.ts.
import { type Target, targetKey } from '@/pages/workspace/model/store';
import { call, type SnapshotJson } from '@/shared/api';
import { failure } from '@/shared/lib/errors';

type Read = {
  controller: AbortController;
  readers: number;
  promise: Promise<SnapshotJson>;
};
const pending = new Map<string, Read>();

export function snapshotMetadata(snapshot: SnapshotJson): SnapshotJson {
  const { drafts: _drafts, activity: _activity, ...metadata } = snapshot;
  return metadata;
}

export function readSnapshot(target: Target, signal: AbortSignal): Promise<SnapshotJson> {
  if (signal.aborted) return Promise.reject(signal.reason);
  const key = targetKey(target);
  let entry = pending.get(key);
  if (!entry) {
    const controller = new AbortController();
    const read: Read = {
      controller,
      readers: 0,
      promise: call(target.workspace, { state: { sessionId: target.session } }, controller.signal)
        .then((reply) => {
          if (!reply.state) throw failure('invalid_response');
          return reply.state;
        })
        .finally(() => {
          if (pending.get(key) === read) pending.delete(key);
        }),
    };
    pending.set(key, read);
    entry = read;
  }
  const read = entry;
  read.readers++;
  return new Promise((resolve, reject) => {
    let closed = false;
    const close = () => {
      if (closed) return;
      closed = true;
      signal.removeEventListener('abort', abort);
      if (--read.readers === 0) {
        read.controller.abort();
        if (pending.get(key) === read) pending.delete(key);
      }
    };
    const abort = () => {
      close();
      reject(signal.reason);
    };
    signal.addEventListener('abort', abort, { once: true });
    void read.promise.then(
      (snapshot) => {
        if (!closed) {
          close();
          resolve(snapshot);
        }
      },
      (error: unknown) => {
        if (!closed) {
          close();
          reject(error);
        }
      }
    );
  });
}
