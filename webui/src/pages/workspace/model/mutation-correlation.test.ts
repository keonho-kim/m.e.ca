// Purpose: Keep local result notifications from duplicating reads without hiding another conversation's change.
// Pattern: Bounded correlation contract tests.
// Usage: bun test src/pages/workspace/model/mutation-correlation.test.ts.
// Related: webui/src/pages/workspace/model/mutation-correlation.ts.
import { afterEach, expect, test } from 'bun:test';
import {
  clearMutationCorrelations,
  consumeMutation,
  rememberMutation,
} from '@/pages/workspace/model/mutation-correlation';

const target = { workspace: 'w', session: 'one' };
afterEach(() => clearMutationCorrelations());
test('a mutation key only acknowledges its own conversation once', () => {
  rememberMutation(target, 'key');
  expect(consumeMutation({ ...target, session: 'other' }, 'key')).toBe(false);
  expect(consumeMutation(target, 'key')).toBe(true);
  expect(consumeMutation(target, 'key')).toBe(false);
  rememberMutation(target, 'cleared');
  clearMutationCorrelations(target);
  expect(consumeMutation(target, 'cleared')).toBe(false);
});
test('missed stream notices retain at most 128 correlation identities', () => {
  for (let i = 0; i < 129; i++) rememberMutation(target, String(i));
  expect(consumeMutation(target, '0')).toBe(false);
  expect(consumeMutation(target, '1')).toBe(true);
  clearMutationCorrelations();
  expect(consumeMutation(target, '128')).toBe(false);
});
