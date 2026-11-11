// Purpose: Own cancellable SSE connections, stale-delivery fences and one reconnect timer.
// Pattern: Concrete browser I/O lifecycle shared by task and file observers.
// Usage: Consumers prepare a fresh URL and provide named event decoders for each connection.
// Related: webui/src/pages/workspace/api/events.ts, webui/src/pages/workspace/model/file-events.ts.
export type StreamConnection = {
  url: string;
  events: Record<string, (data: string) => void>;
};
export function subscribeEventStream(options: {
  connect: (signal: AbortSignal) => Promise<StreamConnection>;
  status: (value: 'connecting' | 'connected' | 'reconnecting') => void;
  failed: (error: unknown) => boolean;
}) {
  const controller = new AbortController();
  const stopped = () => controller.signal.aborted;
  let current: EventSource | undefined,
    retry: ReturnType<typeof setTimeout> | undefined,
    generation = 0;
  const reconnect = () => {
    if (stopped() || retry) return;
    options.status('reconnecting');
    retry = setTimeout(() => {
      retry = undefined;
      void open();
    }, 3000);
  };
  const stopConnection = () => {
    generation++;
    current?.close();
    current = undefined;
  };
  async function open() {
    if (stopped()) return;
    const owned = ++generation;
    options.status('connecting');
    try {
      const connection = await options.connect(controller.signal);
      if (stopped() || generation !== owned) return;
      const source = new EventSource(connection.url);
      current = source;
      const live = () => !stopped() && current === source && generation === owned;
      const failed = (error?: unknown) => {
        if (!live()) return;
        stopConnection();
        if (error === undefined || options.failed(error)) reconnect();
      };
      source.onopen = () => {
        if (live()) options.status('connected');
      };
      source.onerror = () => failed();
      for (const [name, decode] of Object.entries(connection.events)) {
        source.addEventListener(name, (event) => {
          if (!live()) return;
          try {
            decode((event as MessageEvent<string>).data);
          } catch (error) {
            failed(error);
          }
        });
      }
    } catch (error) {
      if (!stopped() && generation === owned && options.failed(error)) reconnect();
    }
  }
  void open();
  return () => {
    controller.abort();
    stopConnection();
    if (retry) clearTimeout(retry);
    retry = undefined;
  };
}
