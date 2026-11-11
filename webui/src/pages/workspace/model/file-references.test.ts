// Purpose: Verify file mutations share the submission barrier and reference failures never skip later work.
// Pattern: Controlled native-command lifecycle regression tests.
// Usage: bun test src/pages/workspace/model/file-references.test.ts.
// Related: webui/src/pages/workspace/model/file-references.ts.
import { afterEach, expect, spyOn, test } from 'bun:test';
import { mutateFile } from '@/pages/workspace/model/file-mutations';
import { refreshReferences, selectReference } from '@/pages/workspace/model/file-references';
import {
  cancelReferenceTarget,
  cancelSelections,
  waitSelections,
} from '@/pages/workspace/model/reference-queue';
import { type Target, targetKey, useWorkspaceStore } from '@/pages/workspace/model/store';
import type { CommandJson } from '@/shared/api';

const originalFetch = globalThis.fetch;
const originalState = useWorkspaceStore.getState();
const target: Target = { workspace: 'files', session: 'session-one' };
function respond(run: (command: CommandJson) => Response | Promise<Response>) {
  globalThis.fetch = Object.assign(
    async (_url: string | URL | Request, options?: RequestInit) => {
      const command: CommandJson = JSON.parse(String(options?.body));
      const response = await run(command);
      const body = await response.json();
      // Echo the server-owned receipt identity for delayed and immediate replies.
      if (body.fileMutation)
        body.fileMutation = {
          key: command.mutateFile?.key ?? command.lookupMutation?.key,
          state: body.fileMutation.referencesReady ? 'applied' : 'partial',
          ...body.fileMutation,
        };
      return Response.json(body, { status: response.status });
    },
    { preconnect: originalFetch.preconnect }
  );
}
afterEach(() => {
  cancelSelections();
  globalThis.fetch = originalFetch;
  useWorkspaceStore.setState(originalState);
});

test('an authoritative later mutation releases uncertainty from an earlier failure', async () => {
  respond(() => Promise.reject(new Error('connection lost')));
  const uncertain = await mutateFile(target, { path: 'old', name: 'next', revision: 'r' }).catch(
    (error: unknown) => error
  );
  expect(uncertain).toBeDefined();
  respond(() =>
    Response.json({ fileMutation: { path: 'final', referencesReady: true, paths: ['final'] } })
  );
  await mutateFile(target, { path: 'next', name: 'final', revision: 'r2' });
  expect(await waitSelections(target)).toBeUndefined();
});

test('send waits through a delayed filesystem reply and consumes one authoritative batch', async () => {
  let received!: () => void;
  let release!: (response: Response) => void;
  const entered = new Promise<void>((resolve) => {
    received = resolve;
  });
  const response = new Promise<Response>((resolve) => {
    release = resolve;
  });
  const calls: CommandJson[] = [];
  respond((command) => {
    calls.push(command);
    received();
    return response;
  });
  useWorkspaceStore.getState().patchInput(target, { paths: ['old/a', 'old/b'] });
  const mutation = mutateFile(target, { path: 'old', name: 'new', revision: 'revision' });
  await entered;
  let sent = false;
  const waiting = waitSelections(target).then(() => {
    sent = true;
  });
  await Promise.resolve();
  expect(sent).toBe(false);
  release(
    Response.json({
      fileMutation: { path: '/project/new', paths: ['new/a', 'new/b'], referencesReady: true },
    })
  );
  await mutation;
  await waiting;
  expect(sent).toBe(true);
  expect(useWorkspaceStore.getState().inputs[targetKey(target)]?.paths).toEqual(['new/a', 'new/b']);
  expect(calls).toHaveLength(1);
  expect(calls[0]?.mutateFile?.sessionId).toBe(target.session);
});

test('one rejected selection does not skip later queued changes and restores only the final snapshot', async () => {
  const attempted: string[] = [];
  let refreshes = 0;
  respond((command) => {
    if (command.selectFile) {
      attempted.push(command.selectFile.path ?? '');
      return Response.json(
        attempted.length === 1 ? { error: { code: 'not_found', message: 'missing' } } : { ok: {} }
      );
    }
    refreshes++;
    return Response.json({ fileSelections: { paths: ['second'] } });
  });
  const first = selectReference(target, 'first', true);
  const second = selectReference(target, 'second', true);
  const results = await Promise.allSettled([first, second]);
  await waitSelections(target);
  expect(results.map((value) => value.status)).toEqual(['rejected', 'fulfilled']);
  expect(attempted).toEqual(['first', 'second']);
  expect(refreshes).toBe(1);
  expect(useWorkspaceStore.getState().inputs[targetKey(target)]?.paths).toEqual(['second']);
});

test('an uncertain applied mutation blocks submission until deliberate reference repair succeeds', async () => {
  respond((command) =>
    Response.json(
      command.mutateFile
        ? {
            fileMutation: {
              path: '/project/new',
              referencesReady: false,
              errorCode: 'persistence_failed',
            },
          }
        : command.selectFile
          ? { ok: {} }
          : { fileSelections: { paths: ['new'] } }
    )
  );
  const mutationError = await mutateFile(target, { path: 'old', name: 'new' }).catch(
    (error: unknown) => error
  );
  expect(mutationError).toMatchObject({ code: 'references_pending' });
  const submissionError = await waitSelections(target).catch((error: unknown) => error);
  expect(submissionError).toMatchObject({ code: 'references_pending' });
  await selectReference(target, 'new', true);
  await waitSelections(target);
  expect(useWorkspaceStore.getState().inputs[targetKey(target)]?.paths).toEqual(['new']);
});

test('different conversations restore their own reference snapshots after one selection fails', async () => {
  const other = { ...target, session: 'session-two' };
  const restored: string[] = [];
  respond((command) => {
    if (command.selectFile)
      return Response.json(
        command.selectFile.sessionId === target.session
          ? { error: { code: 'not_found', message: 'missing' } }
          : { ok: {} }
      );
    const session = command.fileSelections?.sessionId ?? '';
    restored.push(session);
    return Response.json({
      fileSelections: { paths: session === target.session ? [] : ['second'] },
    });
  });
  const first = selectReference(target, 'missing', true);
  const second = selectReference(other, 'second', true);
  await Promise.allSettled([first, second]);
  await waitSelections(target);
  expect(restored).toEqual([target.session, other.session]);
  expect(useWorkspaceStore.getState().inputs[targetKey(target)]?.paths ?? []).toEqual([]);
  expect(useWorkspaceStore.getState().inputs[targetKey(other)]?.paths).toEqual(['second']);
});

test('deleting a target cancels its pending mutation without retaining a blocked target', async () => {
  let received!: () => void, release!: (reply: Response) => void;
  const entered = new Promise<void>((resolve) => {
    received = resolve;
  });
  const reply = new Promise<Response>((resolve) => {
    release = resolve;
  });
  respond(() => {
    received();
    return reply;
  });
  const mutation = mutateFile(target, { path: 'old', name: 'new' });
  await entered;
  cancelReferenceTarget(target);
  release(Response.json({ fileMutation: { path: 'new', paths: ['new'], referencesReady: true } }));
  expect((await Promise.allSettled([mutation]))[0].status).toBe('rejected');
  await waitSelections(target);
});

test('a lost file reply is recovered by receipt lookup without a second mutation', async () => {
  let mutations = 0,
    lookups = 0;
  respond((command) => {
    if (command.mutateFile) {
      mutations++;
      return Promise.reject(new Error('reply lost after rename'));
    }
    lookups++;
    expect(command.lookupMutation?.key).toBeTruthy();
    return Response.json({
      fileMutation: { path: 'renamed', referencesReady: true, paths: ['renamed'] },
    });
  });
  await mutateFile(target, { path: 'original', name: 'renamed' });
  expect(mutations).toBe(1);
  expect(lookups).toBe(1);
  expect(await waitSelections(target)).toBeUndefined();
});

test('deleting a target stops pending receipt polling and releases its timer', async () => {
  const original = setTimeout;
  let armed!: () => void;
  const waiting = new Promise<void>((resolve) => {
    armed = resolve;
  });
  let poll: ReturnType<typeof setTimeout> | undefined;
  const schedule = spyOn(globalThis, 'setTimeout').mockImplementation(((
    handler: () => void,
    delay?: number
  ) => {
    const timer = original(handler, delay);
    if (delay === 500) {
      poll = timer;
      armed();
    }
    return timer;
  }) as typeof setTimeout);
  const clear = spyOn(globalThis, 'clearTimeout');
  try {
    respond((command) =>
      command.mutateFile
        ? Promise.reject(new Error('lost'))
        : Response.json({ fileMutation: { state: 'pending' } })
    );
    const mutation = mutateFile(target, { path: 'before', name: 'after' });
    await waiting;
    cancelReferenceTarget(target);
    const [result] = await Promise.allSettled([mutation]);
    expect(result.status).toBe('rejected');
    expect(clear.mock.calls.some((args) => args[0] === poll)).toBe(true);
    expect(await waitSelections(target)).toBeUndefined();
  } finally {
    schedule.mockRestore();
    clear.mockRestore();
  }
});

test('canceling a notification refresh releases its target without publishing a stale failure', async () => {
  let entered!: () => void, release!: (response: Response) => void;
  const requested = new Promise<void>((resolve) => {
    entered = resolve;
  });
  const response = new Promise<Response>((resolve) => {
    release = resolve;
  });
  respond(() => {
    entered();
    return response;
  });
  const refresh = refreshReferences(target);
  await requested;
  cancelReferenceTarget(target);
  release(Response.json({ fileSelections: { paths: ['stale'] } }));
  await refresh;
  expect(useWorkspaceStore.getState().inputs[targetKey(target)]).toBeUndefined();
  expect(await waitSelections(target)).toBeUndefined();
});
