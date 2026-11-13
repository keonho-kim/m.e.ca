// Purpose: Yield non-urgent preparation to browser input without abandoning cancellation.
// Pattern: Feature-detected native scheduling with an owned timer fallback.
// Usage: Large submission preflight uses this between bounded scan chunks.
// Related: webui/src/pages/workspace/model/submission-limits.ts.
export async function yieldToInput(signal?: AbortSignal) {
  signal?.throwIfAborted();
  const scheduler = (globalThis as unknown as { scheduler?: { yield?: () => Promise<void> } })
    .scheduler;
  if (scheduler?.yield) await scheduler.yield();
  else
    await new Promise<void>((resolve, reject) => {
      const abort = () => {
        clearTimeout(timer);
        reject(signal?.reason);
      };
      const timer = setTimeout(() => {
        signal?.removeEventListener('abort', abort);
        resolve();
      }, 0);
      signal?.addEventListener('abort', abort, { once: true });
    });
  signal?.throwIfAborted();
}
