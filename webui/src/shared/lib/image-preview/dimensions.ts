// Purpose: Inspect common raster dimensions before allocating decoded browser pixels.
// Pattern: Bounded file-header parser with a server fallback for unsupported containers.
// Usage: The preview pipeline never decodes an unclassified image in the browser.
// Related: webui/src/shared/lib/image-preview/index.ts, internal/media/thumbnail/render.go.
export async function imageDimensions(
  file: Blob,
  signal: AbortSignal
): Promise<{ width: number; height: number } | undefined> {
  const header = new DataView(await file.slice(0, 64).arrayBuffer());
  signal.throwIfAborted();
  const word = (at: number, count: number) =>
    Array.from({ length: count }, (_, n) => String.fromCharCode(header.getUint8(at + n))).join('');
  if (header.byteLength >= 24 && header.getUint32(0) === 0x89504e47)
    return { width: header.getUint32(16), height: header.getUint32(20) };
  if (header.byteLength >= 10 && word(0, 3) === 'GIF')
    return { width: header.getUint16(6, true), height: header.getUint16(8, true) };
  if (header.byteLength >= 26 && word(0, 2) === 'BM') {
    if (header.getUint32(14, true) === 12)
      return { width: header.getUint16(18, true), height: header.getUint16(20, true) };
    return { width: header.getInt32(18, true), height: Math.abs(header.getInt32(22, true)) };
  }
  if (header.byteLength >= 30 && word(0, 4) === 'RIFF' && word(8, 4) === 'WEBP') {
    const u24 = (at: number) =>
      header.getUint8(at) + (header.getUint8(at + 1) << 8) + (header.getUint8(at + 2) << 16);
    if (word(12, 4) === 'VP8X') return { width: u24(24) + 1, height: u24(27) + 1 };
    if (word(12, 4) === 'VP8 ')
      return {
        width: header.getUint16(26, true) & 16383,
        height: header.getUint16(28, true) & 16383,
      };
    if (word(12, 4) === 'VP8L') {
      const bits = header.getUint32(21, true);
      return { width: (bits & 16383) + 1, height: ((bits >>> 14) & 16383) + 1 };
    }
  }
  if (header.byteLength < 2 || header.getUint16(0) !== 0xffd8) return;
  let offset = 2;
  for (let segments = 0; segments < 1024 && offset + 9 <= file.size; segments++) {
    const part = new DataView(await file.slice(offset, offset + 12).arrayBuffer());
    signal.throwIfAborted();
    if (part.getUint8(0) !== 255) return;
    const marker = part.getUint8(1);
    if (marker === 255) {
      offset++;
      continue;
    }
    if (marker >= 0xc0 && marker <= 0xcf && ![0xc4, 0xc8, 0xcc].includes(marker))
      return { width: part.getUint16(7), height: part.getUint16(5) };
    if (marker === 0xda || marker === 0xd9) return;
    const bytes = part.getUint16(2);
    if (bytes < 2) return;
    offset += bytes + 2;
  }
}
