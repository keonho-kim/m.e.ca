// Purpose: Compile the embedded Go fixture once for the browser test run.
// Pattern: Playwright global setup owning a temporary executable.
// Usage: Playwright calls prepare before starting test workers.
// Related: webui/e2e/browser-server.ts, cmd/meca/ui_browser_test.go.
import { execFile } from 'node:child_process';
import { mkdtemp, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { promisify } from 'node:util';

export default async function prepare() {
  const directory = await mkdtemp(join(tmpdir(), 'meca-browser-tests-'));
  const binary = join(directory, 'meca.test');
  try {
    await promisify(execFile)(
      'go',
      ['test', '-c', '-tags=godbus_no_tcp', '-o', binary, './cmd/meca'],
      { cwd: fileURLToPath(new URL('../../', import.meta.url)), timeout: 180000 }
    );
    process.env.MECA_UI_TEST_BINARY = binary;
  } catch (cause) {
    await rm(directory, { recursive: true, force: true });
    throw new Error('Could not compile the embedded browser fixture', { cause });
  }
  return async () => {
    delete process.env.MECA_UI_TEST_BINARY;
    await rm(directory, { recursive: true, force: true });
  };
}
