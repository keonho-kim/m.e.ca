// Purpose: Run an isolated MECA server and mock provider for one browser test.
// Pattern: Test-scoped child process and temporary state.
// Usage: The Playwright server fixture calls withBrowserServer.
// Related: webui/e2e/prepare.ts, webui/e2e/fixture.ts, cmd/meca/ui_browser_test.go.
import { spawn } from 'node:child_process';
import { mkdtemp, readFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { createInterface } from 'node:readline';
import { setTimeout as delay } from 'node:timers/promises';

export type BrowserFixture = {
  url: string;
  tokenFile: string;
  projectA: string;
  projectB: string;
  home: string;
  provider: string;
  archiveSession: string;
};

export async function withBrowserServer(use: (server: BrowserFixture) => Promise<void>) {
  const binary = process.env.MECA_UI_TEST_BINARY;
  if (!binary) throw new Error('Playwright did not prepare the MECA test executable');
  const directory = await mkdtemp(join(tmpdir(), 'meca-browser-case-'));
  const descriptor = join(directory, 'connection.json');
  const child = spawn(binary, ['-test.run=^TestUIBrowserFixture$', '-test.timeout=3m'], {
    cwd: directory,
    env: { ...process.env, TMPDIR: directory, MECA_UI_DESCRIPTOR: descriptor },
    stdio: ['ignore', 'pipe', 'pipe'],
  });
  let output = '';
  for (const stream of [child.stdout, child.stderr]) {
    createInterface({ input: stream }).on('line', (line) => {
      if (line.startsWith('Server token:')) return;
      output = `${output}${line}\n`.slice(-8192);
    });
  }
  const running = () =>
    child.pid !== undefined && child.exitCode === null && child.signalCode === null;
  const exit = new Promise<number | null>((resolve) => {
    child.once('error', (error) => {
      output += error.message;
    });
    child.once('close', resolve);
  });

  let code: number | null | undefined;
  try {
    const deadline = Date.now() + 20000;
    let server: BrowserFixture | undefined;
    while (running() && Date.now() < deadline) {
      try {
        server = JSON.parse(await readFile(descriptor, 'utf8'));
        break;
      } catch (error) {
        if ((error as NodeJS.ErrnoException).code !== 'ENOENT') throw error;
        await delay(25);
      }
    }
    if (!server) {
      if (!running()) await exit;
      throw new Error(
        `MECA browser fixture did not become ready. Build the embedded UI first.\n${output}`
      );
    }
    await use(server);
  } finally {
    if (running()) child.kill('SIGTERM');
    // The Go HTTP server can use five seconds to drain before closing its state.
    const deadline = new AbortController();
    try {
      code = await Promise.race([
        exit,
        delay(10000, undefined, { signal: deadline.signal, ref: false }),
      ]);
    } finally {
      deadline.abort();
    }
    if (code === undefined) {
      child.kill('SIGKILL');
      await exit;
    }
    await rm(directory, { recursive: true, force: true });
  }
  if (code !== 0) throw new Error(`MECA browser fixture did not exit cleanly.\n${output}`);
}
