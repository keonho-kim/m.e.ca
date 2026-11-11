// Purpose: Verify explicit logout release and temporary authentication expiry have distinct lifetimes.
// Pattern: Server-session transition tests.
// Usage: bun test src validates session cleanup ownership.
// Related: webui/src/entities/server-session/model/session.ts.
import { afterEach, expect, test } from 'bun:test';
import {
  endSession,
  establishSession,
  reportError,
  useServerSession,
} from '@/entities/server-session/model/session';
import { ApiError, failure } from '@/shared/lib/errors';

afterEach(() => useServerSession.setState(useServerSession.getInitialState(), true));

test('explicit logout advances the mounted workspace lifetime while expiry preserves recovery', () => {
  establishSession('ko');
  const revision = useServerSession.getState().logoutRevision;
  reportError(failure('unauthorized'));
  expect(useServerSession.getState().connected).toBe(false);
  expect(useServerSession.getState().logoutRevision).toBe(revision);
  establishSession();
  endSession();
  expect(useServerSession.getState().logoutRevision).toBe(revision + 1);
  expect(useServerSession.getState().notice).toBe('');
  expect(useServerSession.getState().language).toBe('ko');
});

test('intentional request cancellation does not create a failure notice', () => {
  const before = useServerSession.getState();
  reportError(new DOMException('operation stopped', 'AbortError'));
  expect(useServerSession.getState()).toBe(before);
  reportError(failure('network'));
  expect(useServerSession.getState().notice).toBeTruthy();
  reportError(new ApiError('canceled', 'partial file operation'));
  expect(useServerSession.getState().notice).toBe('partial file operation');
  useServerSession.setState(before);
});
