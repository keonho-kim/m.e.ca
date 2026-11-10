// Purpose: Verify request preflight accounts for UTF-8 and JSON escaping before uploads.
// Pattern: Pure boundary contract tests.
// Usage: bun test src/pages/workspace/model/submission-limits.test.ts.
// Related: webui/src/pages/workspace/model/submission-limits.ts.
import { expect, test } from 'bun:test';
import { checkSubmissionLimits } from '@/pages/workspace/model/submission-limits';
import { emptyInput } from '@/pages/workspace/model/types';
import { failure } from '@/shared/lib/errors';

const limits = { promptBytes: '1500', commandBytes: '2000' };
test('valid UTF-8 fits the prompt while oversized logical text is rejected', () => {
  expect(() =>
    checkSubmissionLimits({ ...emptyInput(), text: '한'.repeat(500) }, limits)
  ).not.toThrow();
  expect(() => checkSubmissionLimits({ ...emptyInput(), text: '😀'.repeat(376) }, limits)).toThrow(
    failure('input_limit')
  );
});
test('JSON escaping can exceed the wire limit while the same raw input size fits', () => {
  expect(() =>
    checkSubmissionLimits({ ...emptyInput(), text: 'x'.repeat(1000) }, limits)
  ).not.toThrow();
  expect(() => checkSubmissionLimits({ ...emptyInput(), text: '"'.repeat(1000) }, limits)).toThrow(
    failure('input_limit')
  );
});
