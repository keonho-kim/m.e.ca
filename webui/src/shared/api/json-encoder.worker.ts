// Purpose: Serialize large command bodies away from the browser input/rendering thread.
// Pattern: Single-message worker returning a transferable UTF-8 buffer.
// Usage: json-body owns worker creation, cancellation and termination.
// Related: webui/src/shared/api/json-body.ts.
const scope = self as unknown as {
  onmessage: ((event: MessageEvent<unknown>) => void) | null;
  postMessage: (value: unknown, transfer?: Transferable[]) => void;
};
scope.onmessage = (event) => {
  try {
    const bytes = new TextEncoder().encode(JSON.stringify(event.data));
    scope.postMessage({ bytes: bytes.buffer }, [bytes.buffer]);
  } catch {
    scope.postMessage({ failed: true });
  }
};

export {};
