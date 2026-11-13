// Purpose: Identify browser source inputs consistently in compiled code and embedded metadata.
// Pattern: Deterministic build fingerprint without Git or private environment values.
// Usage: vite.config evaluates this before compiling the browser application.
// Related: internal/webui/assets.go, webui/src/shared/ui/recovery/index.tsx.
import { createHash } from 'node:crypto';
import { readdirSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('..', import.meta.url));
function inputs(path: string): string[] {
  return readdirSync(resolve(root, path), { withFileTypes: true }).flatMap((entry) =>
    entry.isDirectory() ? inputs(`${path}/${entry.name}`) : [`${path}/${entry.name}`]
  );
}
export function browserBuildID() {
  const hash = createHash('sha256');
  for (const file of [
    ...inputs('src').filter((path) => !path.endsWith('.test.ts')),
    'package.json',
    'bun.lock',
    'index.html',
    'vite.config.ts',
    'scripts/build-id.ts',
  ].sort()) {
    hash.update(file);
    hash.update('\0');
    hash.update(readFileSync(resolve(root, file)));
  }
  return hash.digest('hex');
}
