// Purpose: Verify staged-image reply parsing and bounded cleanup of unused HTTP response bodies.
// Pattern: Deterministic browser transport contracts with a controlled fetch boundary.
// Usage: bun test src/shared/api/client.test.ts.
// Related: webui/src/shared/api/client.ts, webui/src/shared/lib/errors/index.ts.
import { expect, spyOn, test } from 'bun:test';
import { control, releaseUpload, upload } from '@/shared/api/client';

test('control failures use the safe network catalog and unauthorized bodies are released', async () => {
  const request = spyOn(globalThis, 'fetch');
  try {
    request.mockRejectedValueOnce(new TypeError('private transport detail'));
    expect(await control('project', 'task', 'cancel').catch((error) => error)).toMatchObject({
      code: 'network',
    });
    let cancelled = false;
    request.mockResolvedValueOnce(
      new Response(
        new ReadableStream({
          cancel: () => {
            cancelled = true;
          },
        }),
        { status: 401 }
      )
    );
    expect(await control('project', 'task', 'cancel').catch((error) => error)).toMatchObject({
      code: 'unauthorized',
    });
    expect(cancelled).toBe(true);
    expect(request.mock.calls).toHaveLength(2);
  } finally {
    request.mockRestore();
  }
});

test('image uploads reject malformed and empty identifiers before a start request can use them', async () => {
  const request = spyOn(globalThis, 'fetch');
  try {
    for (const value of [{}, { id: 3 }, { id: '' }, null]) {
      request.mockResolvedValueOnce(Response.json(value));
      const result = await upload('project', new File(['pixels'], 'screen.png')).catch(
        (error) => error
      );
      expect(result).toMatchObject({ code: 'invalid_response' });
    }
    request.mockResolvedValueOnce(new Response('<html>offline</html>'));
    const invalid = await upload('project', new File(['pixels'], 'screen.png')).catch(
      (error) => error
    );
    expect(invalid).toMatchObject({ code: 'invalid_response' });
    request.mockResolvedValueOnce(Response.json({ id: 'img_valid' }));
    expect(await upload('project', new File(['pixels'], 'screen.png'))).toEqual({
      id: 'img_valid',
    });
  } finally {
    request.mockRestore();
  }
});

test('staged-image deletion has an owned timeout and cancels its unneeded response body', async () => {
  let cancelled = false;
  const request = spyOn(globalThis, 'fetch').mockResolvedValue(
    new Response(
      new ReadableStream({
        cancel: () => {
          cancelled = true;
        },
      })
    )
  );
  try {
    await releaseUpload('project', 'img_staged');
    expect(request.mock.calls[0]?.[1]?.signal).toBeInstanceOf(AbortSignal);
    expect(request.mock.calls[0]?.[1]?.method).toBe('DELETE');
    expect(cancelled).toBe(true);
  } finally {
    request.mockRestore();
  }
});
