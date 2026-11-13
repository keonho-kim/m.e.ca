// Purpose: Bound large JSON serialization and keep command controls off its work queue.
// Pattern: Single-worker admission with abortable queued ownership and transferable output.
// Usage: Native HTTP serializes small envelopes inline and delegates large text bodies here.
// Related: webui/src/shared/api/json-encoder.worker.ts, webui/src/shared/api/http.ts.
import { failure } from '@/shared/lib/errors';

type Job = {
  value: unknown;
  bytes: number;
  signal?: AbortSignal | null;
  resolve: (body: Uint8Array<ArrayBuffer>) => void;
  reject: (error: unknown) => void;
  dispose: () => void;
};
const waiting: Job[] = [];
let active: Job | undefined;
let held = 0;
function estimate(value: unknown, depth = 0): number {
  if (typeof value === 'string') return value.length * 2;
  if (!value || typeof value !== 'object' || depth > 8) return 16;
  return Object.values(value).reduce<number>(
    (total, part) => total + estimate(part, depth + 1),
    32
  );
}
function dispatch() {
  if (active) return;
  const job = waiting.shift();
  if (!job) return;
  active = job;
  let worker: Worker | undefined;
  const finish = (error?: unknown, bytes?: ArrayBuffer) => {
    if (active !== job) return;
    worker?.terminate();
    job.signal?.removeEventListener('abort', abort);
    job.dispose();
    held -= job.bytes;
    active = undefined;
    if (bytes) job.resolve(new Uint8Array(bytes));
    else job.reject(error ?? failure('invalid_response'));
    dispatch();
  };
  const abort = () => finish(job.signal?.reason);
  job.dispose();
  job.signal?.addEventListener('abort', abort, { once: true });
  if (job.signal?.aborted) {
    abort();
    return;
  }
  try {
    worker = new Worker(new URL('./json-encoder.worker.ts', import.meta.url), { type: 'module' });
    worker.onmessage = (event) => {
      const value: unknown = event.data;
      if (
        value &&
        typeof value === 'object' &&
        'bytes' in value &&
        value.bytes instanceof ArrayBuffer
      )
        finish(undefined, value.bytes);
      else finish(failure('invalid_response'));
    };
    worker.onerror = () => finish(failure('invalid_response'));
    worker.postMessage(job.value);
    job.value = undefined;
  } catch {
    finish(failure('invalid_response'));
  }
}
export async function jsonBody(value: unknown, signal?: AbortSignal | null): Promise<BodyInit> {
  signal?.throwIfAborted();
  const bytes = estimate(value);
  if (bytes < 512 * 1024 || typeof Worker === 'undefined' || typeof document === 'undefined')
    return JSON.stringify(value);
  if (held + bytes > 64 * 1024 * 1024) throw failure('queue_limit');
  held += bytes;
  return new Promise<Uint8Array<ArrayBuffer>>((resolve, reject) => {
    const job: Job = { value, bytes, signal, resolve, reject, dispose: () => {} };
    const abort = () => {
      const index = waiting.indexOf(job);
      if (index < 0) return;
      waiting.splice(index, 1);
      held -= bytes;
      job.dispose();
      job.value = undefined;
      reject(signal?.reason);
    };
    job.dispose = () => signal?.removeEventListener('abort', abort);
    signal?.addEventListener('abort', abort, { once: true });
    waiting.push(job);
    dispatch();
  });
}
