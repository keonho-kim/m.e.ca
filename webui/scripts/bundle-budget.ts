// Purpose: Check compressed browser budgets and optionally compare Brotli's distribution cost.
// Pattern: Development-only measurement over built source assets and the packager's gzip cache.
// Usage: bun run bundle:budget [--brotli]; no extra formats enter the release binary.
// Related: webui/vite.config.ts, devtools/cmd/uiassets/compression.go.
import { createHash } from 'node:crypto';
import { readdir, readFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { promisify } from 'node:util';
import { brotliCompress, constants, gzipSync } from 'node:zlib';

const compress = promisify(brotliCompress);
const root = resolve(import.meta.dir, '..');
const html = await readFile(resolve(root, 'dist/index.html'), 'utf8');
const initial = new Set(
  [...html.matchAll(/(?:src|href)="\/ui\/(assets\/[^"]+)"/g)].map((match) => match[1])
);
const report = {
  initialRaw: 0,
  initialGzip: 0,
  totalGzip: 0,
  initialBrotli: 0,
  additionalBrotliBytes: 0,
};
async function walk(path: string) {
  for (const entry of await readdir(resolve(root, 'dist', path), { withFileTypes: true })) {
    const name = path ? `${path}/${entry.name}` : entry.name;
    if (entry.isDirectory()) {
      await walk(name);
      continue;
    }
    const body = await readFile(resolve(root, 'dist', name));
    const hash = createHash('sha256').update(body).digest('hex');
    const gzip = await readFile(resolve(root, '.cache/assets', `${hash}.gz`)).catch(() =>
      gzipSync(body, { level: 9 })
    );
    report.totalGzip += gzip.length;
    if (initial.has(name)) {
      report.initialRaw += body.length;
      report.initialGzip += gzip.length;
    }
    if (process.argv.includes('--brotli')) {
      const compressed = await compress(body, { params: { [constants.BROTLI_PARAM_QUALITY]: 11 } });
      report.additionalBrotliBytes += compressed.length;
      if (initial.has(name)) report.initialBrotli += compressed.length;
    }
  }
}
await walk('');
console.log(JSON.stringify(report, null, 2));
if (report.initialGzip > 320 * 1024 || report.totalGzip > 1536 * 1024) process.exitCode = 1;
