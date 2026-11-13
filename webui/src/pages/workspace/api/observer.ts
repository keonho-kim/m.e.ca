// Purpose: Multiplex browser observers through one cookie-authenticated WebSocket per page.
// Pattern: Reference-counted connection with fenced reconnect and page lifecycle ownership.
// Usage: Task and file observers prepare fresh subscriptions; commands remain ordinary HTTP.
// Related: internal/serverapi/ui_observer.go, webui/src/pages/workspace/api/events.ts.
import { nativeJSON } from '@/shared/api';
import { ApiError, failure } from '@/shared/lib/errors';

type StreamConnection = {
  url: string;
  events: Record<string, (data: unknown) => void>;
};
type Options = {
  connect: (signal: AbortSignal) => Promise<StreamConnection>;
  status: (value: 'connecting' | 'connected' | 'reconnecting') => void;
  failed: (error: unknown) => boolean;
};
type Subscriber = Options & {
  id: string;
  controller?: AbortController;
  connection?: StreamConnection;
};
const subscribers = new Map<string, Subscriber>();
let socket: WebSocket | undefined;
let retry: ReturnType<typeof setTimeout> | undefined;
let opening: AbortController | undefined;
let handshake: ReturnType<typeof setTimeout> | undefined;
let suspended = false;
let attempts = 0;
let sequence = 0;

function stopConnection() {
  if (handshake) clearTimeout(handshake);
  handshake = undefined;
  opening?.abort();
  opening = undefined;
  if (retry) clearTimeout(retry);
  retry = undefined;
  const previous = socket;
  socket = undefined;
  previous?.close();
  for (const value of subscribers.values()) {
    value.controller?.abort();
    value.controller = undefined;
    value.connection = undefined;
  }
}
function reconnect(error: unknown = failure('network')) {
  stopConnection();
  if (suspended || !subscribers.size) return;
  for (const value of subscribers.values()) {
    if (value.failed(error)) value.status('reconnecting');
    else subscribers.delete(value.id);
  }
  if (!subscribers.size) {
    listen(false);
    return;
  }
  // Browser observer retries are independent of the provider's retry policy.
  const ceiling = Math.min(5000, 1000 * ++attempts);
  retry = setTimeout(
    () => {
      retry = undefined;
      void open();
    },
    ceiling * (0.75 + Math.random() * 0.5)
  );
}
async function prepare(value: Subscriber, owner: WebSocket) {
  value.controller?.abort();
  const controller = new AbortController();
  value.controller = controller;
  value.status('connecting');
  try {
    const connection = await value.connect(
      AbortSignal.any([controller.signal, AbortSignal.timeout(15000)])
    );
    if (controller.signal.aborted || socket !== owner || !subscribers.has(value.id)) return;
    value.connection = connection;
    owner.send(JSON.stringify({ id: value.id, url: connection.url }));
    attempts = 0;
    value.status('connected');
  } catch (error) {
    if (!controller.signal.aborted && socket === owner) {
      if (value.failed(error)) reconnect();
      else {
        subscribers.delete(value.id);
        if (!subscribers.size) {
          stopConnection();
          listen(false);
        }
      }
    }
  }
}
async function open() {
  if (socket || opening || suspended || !subscribers.size) return;
  const controller = new AbortController();
  opening = controller;
  try {
    const { body } = await nativeJSON('/v1/ui/session', {
      signal: AbortSignal.any([controller.signal, AbortSignal.timeout(10000)]),
    });
    if (controller.signal.aborted) return;
    if (!body || typeof body !== 'object' || !('authenticated' in body))
      throw failure('invalid_response');
    if (body.authenticated !== true) throw failure('unauthorized');
    const url = new URL(
      '/v1/ui/events',
      typeof location === 'undefined' ? 'http://localhost' : location.href
    );
    url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
    const owner = new WebSocket(url);
    socket = owner;
    handshake = setTimeout(() => {
      if (socket === owner) reconnect(failure('read_timeout'));
    }, 10000);
    owner.onopen = () => {
      if (socket !== owner) return;
      if (handshake) clearTimeout(handshake);
      handshake = undefined;
      for (const value of subscribers.values()) void prepare(value, owner);
    };
    owner.onmessage = (event) => {
      if (socket !== owner) return;
      try {
        const frame: unknown = JSON.parse(String(event.data));
        if (
          !frame ||
          typeof frame !== 'object' ||
          !('id' in frame) ||
          typeof frame.id !== 'string' ||
          !('messages' in frame) ||
          !Array.isArray(frame.messages) ||
          frame.messages.length > 128
        )
          throw failure('invalid_response');
        const value = subscribers.get(frame.id);
        if (!value?.connection) return;
        for (const item of frame.messages) {
          if (
            !item ||
            typeof item !== 'object' ||
            typeof item.event !== 'string' ||
            !('payload' in item)
          )
            throw failure('invalid_response');
          if (item.event === 'error') {
            const error = new ApiError(
              item.payload.code ?? 'unavailable',
              item.payload.message ?? ''
            );
            if (value.failed(error)) reconnect(error);
            else {
              subscribers.delete(value.id);
              value.controller?.abort();
              value.connection = undefined;
              owner.send(JSON.stringify({ id: value.id }));
              if (!subscribers.size) {
                stopConnection();
                listen(false);
              }
            }
            return;
          }
          value.connection.events[item.event]?.(item.payload);
        }
      } catch {
        reconnect(failure('invalid_response'));
      } finally {
        if (socket === owner && owner.readyState === WebSocket.OPEN)
          owner.send(JSON.stringify({ id: 'ack', ack: true }));
      }
    };
    owner.onerror = () => {
      if (socket === owner) reconnect();
    };
    owner.onclose = () => {
      if (socket === owner) reconnect();
    };
  } catch (error) {
    if (!controller.signal.aborted) reconnect(error);
  } finally {
    if (opening === controller) opening = undefined;
  }
}
function hide() {
  suspended = true;
  stopConnection();
}
function resume() {
  suspended = false;
  void open();
}
function listen(enabled: boolean) {
  if (typeof window === 'undefined') return;
  const method = enabled ? 'addEventListener' : 'removeEventListener';
  window[method]('pagehide', hide);
  window[method]('pageshow', resume);
  window[method]('offline', hide);
  window[method]('online', resume);
}
export function subscribeEventStream(options: Options) {
  const first = !subscribers.size;
  const value: Subscriber = { ...options, id: `observer_${++sequence}` };
  subscribers.set(value.id, value);
  if (first) {
    suspended = typeof navigator !== 'undefined' && navigator.onLine === false;
    listen(true);
  }
  if (socket?.readyState === WebSocket.OPEN) void prepare(value, socket);
  else void open();
  return () => {
    value.controller?.abort();
    value.connection = undefined;
    if (subscribers.delete(value.id) && socket?.readyState === WebSocket.OPEN)
      socket.send(JSON.stringify({ id: value.id }));
    if (!subscribers.size) {
      stopConnection();
      attempts = 0;
      listen(false);
    }
  };
}
