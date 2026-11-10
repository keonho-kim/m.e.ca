// Purpose: Package one content-addressed gzip copy of each built browser asset.
// Pattern: Reproducible build adapter.
// Usage: bun run build calls this after Vite.
// Related: internal/webui/assets.go, webui/vite.config.ts.

import { createHash } from 'node:crypto';
import { mkdir, readdir, unlink } from 'node:fs/promises';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

const output = fileURLToPath(new URL('../../internal/webui/assets/', import.meta.url));
await mkdir(output, { recursive: true });
for (const name of await readdir(output))
  if (/^[a-f0-9]{64}\.gz$/.test(name)) await unlink(join(output, name));
const manifest: Record<string, { file: string; type: string; etag: string; bytes: number }> = {};
async function visit(directory: string, prefix = '') {
  for (const entry of await readdir(directory, { withFileTypes: true })) {
    const path = join(directory, entry.name),
      name = prefix + entry.name;
    if (entry.isDirectory()) {
      await visit(path, name + '/');
      continue;
    }
    const source = new Uint8Array(await Bun.file(path).arrayBuffer());
    const hash = createHash('sha256').update(source).digest('hex'),
      file = hash + '.gz';
    await Bun.write(join(output, file), Bun.gzipSync(source, { level: 9 }));
    manifest[name] = {
      file,
      type: Bun.file(path).type,
      etag: 'W/"' + hash + '"',
      bytes: source.byteLength,
    };
  }
}
await visit(fileURLToPath(new URL('../dist/', import.meta.url)));
await Bun.write(join(output, 'manifest.json'), JSON.stringify(manifest));
console.log('Embedded assets:', Object.keys(manifest).length);
