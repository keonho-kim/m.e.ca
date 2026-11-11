// Purpose: Preserve readable tool values and decode only bounded public argument metadata.
// Pattern: Pure presentation boundary regression tests.
// Usage: bun test src/pages/workspace/model/tool-content.test.ts.
// Related: webui/src/pages/workspace/model/tool-content.ts.
import { expect, test } from 'bun:test';
import {
  parseToolValue,
  toolArguments,
  toolFields,
  valueText,
} from '@/pages/workspace/model/tool-content';

test('tool values unwrap JSON strings and preserve multiline content and invalid page fragments', () => {
  expect(parseToolValue('"line one\\nline two"')).toBe('line one\nline two');
  expect(parseToolValue('{"stdout":"unfinished')).toBe('{"stdout":"unfinished');
  expect(valueText({ exit_code: 0 })).toBe('{\n  "exit_code": 0\n}');
  expect(valueText(undefined)).toBe('');
});
test('tool arguments decode Unicode without retaining malformed metadata', () => {
  const args = { command: 'printf 한글' };
  const bytes = new TextEncoder().encode(JSON.stringify(args));
  const decoded = toolArguments(btoa(String.fromCharCode(...bytes)));
  expect(decoded).toEqual(args);
  expect(toolArguments('!invalid-base64')).toBeUndefined();
});

test('command output leads while execution identifiers remain in secondary metadata', () => {
  const value = {
    CommandID: 'command',
    Stdout: 'result',
    Stderr: '',
    ExitCode: 0,
    Truncated: false,
  };
  const view = toolFields(value, false);
  expect(view.content).toEqual([['Stdout', 'result']]);
  expect(view.metadata).toEqual([
    ['CommandID', 'command'],
    ['ExitCode', 0],
    ['Truncated', false],
  ]);
  expect(toolFields(value, true).content).toHaveLength(5);
});
