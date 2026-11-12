// Purpose: Keep remote folder selection in the existing modal without contacting a desktop endpoint.
// Pattern: HTTP boundary admission contract.
// Usage: bun test src/pages/workspace/api/folder-picker.test.ts.
// Related: webui/src/pages/workspace/api/folder-picker.ts.
import { expect, spyOn, test } from 'bun:test';
import { nativeFolder } from '@/pages/workspace/api/folder-picker';

test('remote hostnames and LAN addresses always use the folder modal', async () => {
  for (const hostname of ['meca.example.com', '192.168.0.5', '0.0.0.0', 'evil.localhost']) {
    // A relative native fetch cannot run in this non-browser test process.
    expect(await nativeFolder(hostname, new AbortController().signal)).toEqual({
      kind: 'fallback',
    });
  }
});

test('unsupported desktops fall back while native failures retain their diagnosis', async () => {
  const fetch = spyOn(globalThis, 'fetch');
  try {
    for (const status of [403, 503]) {
      fetch.mockResolvedValueOnce(new Response('', { status }));
      expect(await nativeFolder('localhost', new AbortController().signal)).toEqual({
        kind: 'fallback',
      });
    }
    for (const [status, code] of [
      [409, 'folder_picker_busy'],
      [500, 'folder_picker'],
      [504, 'folder_picker_timeout'],
      [401, 'unauthorized'],
    ] as const) {
      fetch.mockResolvedValueOnce(new Response('', { status }));
      const result = await nativeFolder('localhost', new AbortController().signal).catch(
        (error: unknown) => error
      );
      expect(result).toMatchObject({ code });
    }
    fetch.mockResolvedValueOnce(new Response('{invalid', { status: 200 }));
    const invalid = await nativeFolder('localhost', new AbortController().signal).catch(
      (error: unknown) => error
    );
    expect(invalid).toMatchObject({ code: 'invalid_response' });
  } finally {
    fetch.mockRestore();
  }
});
