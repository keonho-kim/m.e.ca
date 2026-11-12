// Purpose: Verify authentication expiry permits explicit reconnection and cancellation stays quiet.
// Pattern: Server-session transition tests.
// Usage: bun test src validates session cleanup ownership.
// Related: webui/src/entities/server-session/model/session.ts.
import { afterEach, expect, test } from 'bun:test';
import {
  establishSession,
  reportError,
  useServerSession,
} from '@/entities/server-session/model/session';
import { ApiError, failure } from '@/shared/lib/errors';

afterEach(() => useServerSession.setState(useServerSession.getInitialState(), true));

test('authentication expiry permits reconnection without losing the selected language', () => {
  establishSession('ko');
  reportError(failure('unauthorized'));
  expect(useServerSession.getState().connected).toBe(false);
  establishSession();
  expect(useServerSession.getState().connected).toBe(true);
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
