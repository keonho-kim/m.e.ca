// Purpose: Invoke the standard-library packager for one atomic embedded browser bundle.
// Pattern: Source-build command adapter.
// Usage: bun run build calls this after Vite completes.
// Related: devtools/cmd/uiassets/main.go, internal/webui/bundle.go.
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('../../', import.meta.url));
const packager = Bun.spawn(
  [
    'go',
    '-C',
    join(root, 'devtools'),
    'run',
    './cmd/uiassets',
    '-input',
    join(root, 'webui/dist'),
    '-output',
    join(root, 'internal/webui/bundle_generated.go'),
    '-cache',
    join(root, 'webui/.cache/assets'),
  ],
  { stdout: 'inherit', stderr: 'inherit' }
);
const status = await packager.exited;
if (status !== 0) process.exit(status);
