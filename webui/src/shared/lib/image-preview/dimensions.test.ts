// Purpose: Reject oversized pixels before decoding and read common raster headers safely.
// Pattern: Deterministic header and preview-admission contracts.
// Usage: bun test src/shared/lib/image-preview/dimensions.test.ts.
// Related: webui/src/shared/lib/image-preview/dimensions.ts, webui/src/shared/lib/image-preview/index.ts.
import { expect, test } from 'bun:test';
import { imagePreview } from '@/shared/lib/image-preview';
import { imageDimensions } from '@/shared/lib/image-preview/dimensions';

function png(width: number, height: number) {
  const bytes = new Uint8Array(24),
    header = new DataView(bytes.buffer);
  header.setUint32(0, 0x89504e47);
  header.setUint32(16, width);
  header.setUint32(20, height);
  return new File([bytes], 'fixture.png', { type: 'image/png' });
}
test('pixel admission happens before a browser decoder or network request', async () => {
  const file = png(100000, 100000),
    signal = new AbortController().signal;
  expect(await imageDimensions(file, signal)).toEqual({ width: 100000, height: 100000 });
  const error = await imagePreview(
    file,
    { imagePixels: '67108864', previewDecodePixels: '8388608', thumbnailEdge: 256 },
    signal
  ).catch((error: unknown) => error);
  expect(error).toMatchObject({ code: 'image_pixels' });
});
test('short and unclassified headers take the guarded fallback', async () => {
  const signal = new AbortController().signal;
  expect(await imageDimensions(new Blob([]), signal)).toBeUndefined();
  expect(await imageDimensions(new Blob(['not an image']), signal)).toBeUndefined();
  expect(await imageDimensions(png(64, 48), signal)).toEqual({ width: 64, height: 48 });
});
