// Purpose: Cancel protected browser I/O when authentication ends without changing network state.
// Pattern: One authentication generation and abort owner.
// Usage: The session entity changes access; HTTP reads borrow its cancellation signal.
// Related: webui/src/shared/api/http.ts, webui/src/entities/server-session/model/session.ts.
let allowed = true;
let owner = new AbortController();
export function requestAccessAllowed() {
  return allowed;
}
export function requestAccessSignal() {
  return owner.signal;
}
export function setRequestAccess(value: boolean) {
  if (value === allowed) return;
  owner.abort();
  owner = new AbortController();
  allowed = value;
}
