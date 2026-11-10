// Purpose: Verify login accepts the displayed token and distinguishes trust and browser-cookie failures.
// Pattern: Authentication HTTP boundary regression tests.
// Usage: bun test src/features/server-access/api/access.test.ts.
// Related: webui/src/features/server-access/api/access.ts, internal/serverapi/ui_auth.go.
import { afterEach, expect, test } from 'bun:test';
import { login } from '@/features/server-access/api/access';

const originalFetch = globalThis.fetch;
afterEach(() => {
  globalThis.fetch = originalFetch;
});

test('the complete startup token line can be pasted into login', async () => {
  const token = 'test-server-token-for-browser-authentication';
  let authorization = '';
  globalThis.fetch = (async (_input: unknown, init?: RequestInit) => {
    if (init?.method === 'POST') {
      authorization = new Headers(init.headers).get('Authorization') ?? '';
      return new Response(null, { status: 204 });
    }
    return Response.json({ authenticated: true });
  }) as typeof fetch;
  await login(`Server token: ${token}\n`);
  expect(authorization).toBe(`Bearer ${token}`);
});

test('a successful token exchange must establish a browser session', async () => {
  globalThis.fetch = (async (_input: unknown, init?: RequestInit) =>
    init?.method === 'POST'
      ? new Response(null, { status: 204 })
      : Response.json({ authenticated: false })) as typeof fetch;
  const rejected = await login('test-server-token-for-browser-authentication').then(
    () => null,
    (error) => error as unknown
  );
  expect(rejected).toMatchObject({
    code: 'browser_cookie',
  });
});

test('a rejected origin is not reported as an incorrect token', async () => {
  globalThis.fetch = (async (_input: unknown, _init?: RequestInit) =>
    new Response(null, { status: 403 })) as typeof fetch;
  const rejected = await login('test-server-token-for-browser-authentication').then(
    () => null,
    (error) => error as unknown
  );
  expect(rejected).toMatchObject({
    code: 'forbidden',
  });
});
