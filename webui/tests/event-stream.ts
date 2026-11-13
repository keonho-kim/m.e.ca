// Purpose: Drive the browser observer wire contract without real sockets or credentials.
// Pattern: Deterministic WebSocket and authentication fixture.
// Usage: Native stream and workspace observer tests install and restore these adapters.
// Related: webui/src/pages/workspace/api/observer.ts.
import { setRequestAccess } from '@/shared/api/access-lifetime';
export class TestSocket {
  static OPEN = 1;
  static ready: (socket: TestSocket) => void = () => {};
  static all: TestSocket[] = [];
  readyState = 1;
  closed = false;
  id = '';
  onopen: (() => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: (() => void) | null = null;
  onmessage: ((event: MessageEvent<string>) => void) | null = null;
  sent: { id: string; url?: string }[] = [];
  constructor() {
    TestSocket.all.push(this);
    queueMicrotask(() => this.onopen?.());
  }
  send(body: string) {
    const frame = JSON.parse(body) as { id: string; url?: string };
    this.sent.push(frame);
    if (frame.url) {
      this.id = frame.id;
      TestSocket.ready(this);
    }
  }
  close() {
    this.closed = true;
    this.readyState = 3;
    this.onclose?.();
  }
  dispatchEvent(event: MessageEvent<string>) {
    this.onmessage?.(
      new MessageEvent('message', {
        data: JSON.stringify({
          id: this.id,
          messages: [{ event: event.type, payload: JSON.parse(event.data) }],
        }),
      })
    );
    return true;
  }
}
export function streamFetch(
  fetcher: (url: string | URL | Request, options?: RequestInit) => Promise<Response>
) {
  setRequestAccess(true);
  return Object.assign(
    (url: string | URL | Request, options?: RequestInit) =>
      String(url) === '/v1/ui/session'
        ? Promise.resolve(Response.json({ authenticated: true }))
        : fetcher(url, options),
    { preconnect: globalThis.fetch.preconnect }
  );
}
export const settleStream = async () => {
  for (let i = 0; i < 24; i++) await Promise.resolve();
};
