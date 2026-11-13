// Purpose: Produce small static preview blobs while preserving original vision inputs.
// Pattern: Serial browser decode queue with weak derived caching and cancellable server fallback.
// Usage: Composer image cards own object URLs; this module never stores originals or URLs.
// Related: webui/src/shared/lib/image-preview/dimensions.ts, internal/serverapi/ui_thumbnail.go.
import { type CapabilitiesJson, nativeRequest } from '@/shared/api';
import { failure } from '@/shared/lib/errors';
import { imageDimensions } from '@/shared/lib/image-preview/dimensions';

const cache = new WeakMap<File, WeakRef<Blob>>();
let tail: Promise<unknown> = Promise.resolve();
export function imagePreview(
  file: File,
  caps: CapabilitiesJson,
  signal: AbortSignal
): Promise<Blob> {
  const run = async () => {
    signal.throwIfAborted();
    const cached = cache.get(file)?.deref();
    if (cached) return cached;
    const pixels = Number(caps.imagePixels),
      localPixels = Number(caps.previewDecodePixels),
      edge = Number(caps.thumbnailEdge);
    if (!(pixels > 0 && localPixels > 0 && edge > 0 && edge <= 1024))
      throw failure('invalid_response');
    const size = await imageDimensions(file, signal);
    if (
      size &&
      (!Number.isSafeInteger(size.width * size.height) ||
        size.width < 1 ||
        size.height < 1 ||
        size.width * size.height > pixels)
    )
      throw failure('image_pixels');
    let blob: Blob | undefined;
    if (
      size &&
      size.width * size.height <= localPixels &&
      typeof createImageBitmap === 'function'
    ) {
      const scale = Math.min(1, edge / Math.max(size.width, size.height));
      try {
        const bitmap = await createImageBitmap(file, {
          resizeWidth: Math.max(1, Math.round(size.width * scale)),
          resizeHeight: Math.max(1, Math.round(size.height * scale)),
          resizeQuality: 'medium',
        });
        const canvas = document.createElement('canvas');
        try {
          signal.throwIfAborted();
          const bounded = Math.min(1, edge / Math.max(bitmap.width, bitmap.height));
          canvas.width = Math.max(1, Math.round(bitmap.width * bounded));
          canvas.height = Math.max(1, Math.round(bitmap.height * bounded));
          const context = canvas.getContext('2d');
          if (!context) throw failure('image_preview');
          context.drawImage(bitmap, 0, 0, canvas.width, canvas.height);
          blob = await new Promise<Blob>((resolve, reject) =>
            canvas.toBlob(
              (value) => (value ? resolve(value) : reject(failure('image_preview'))),
              'image/png'
            )
          );
        } finally {
          canvas.width = canvas.height = 0;
          bitmap.close();
        }
      } catch {
        signal.throwIfAborted();
      }
    }
    if (!blob) {
      const response = await nativeRequest('/v1/ui/thumbnail', {
        method: 'POST',
        headers: { 'Content-Type': file.type },
        body: file,
        signal: AbortSignal.any([signal, AbortSignal.timeout(30000)]),
      });
      if (!response.ok) {
        await response.body?.cancel();
        throw failure('image_preview');
      }
      blob = await response.blob();
    }
    signal.throwIfAborted();
    if (blob.type !== 'image/png' || blob.size > 1024 * 1024) throw failure('image_preview');
    cache.set(file, new WeakRef(blob));
    return blob;
  };
  const pending = tail.then(run);
  tail = pending.then(
    () => undefined,
    () => undefined
  );
  return pending;
}
