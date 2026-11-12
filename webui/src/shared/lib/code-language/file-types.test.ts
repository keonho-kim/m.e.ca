// Purpose: Verify language/framework filenames share the correct syntax and file-icon identity.
// Pattern: File-type catalog behavior tests.
// Usage: bun test src/shared/lib/code-language.
// Related: webui/src/shared/lib/code-language/file-types.ts, webui/src/shared/ui/file-icon/index.tsx.
import { expect, test } from 'bun:test';
import { codeLanguage, fileType, fileTypes } from '@/shared/lib/code-language';

test('at least thirty language and framework types have a syntax and icon identity', () => {
  expect(Object.keys(fileTypes).length).toBeGreaterThanOrEqual(30);
  for (const entry of Object.values(fileTypes))
    for (const field of entry) expect(field.length).toBeGreaterThan(0);
});
test('prototype-like filenames are ordinary unknown files', () => {
  for (const path of ['constructor', 'file.constructor', '__proto__', 'file.__proto__'])
    expect(fileType(path)).toBeUndefined();
});
test('framework and extension precedence identifies actual files rather than parent-directory names', () => {
  for (const [path, type, syntax] of [
    ['src/App.vue', 'vue', 'vue'],
    ['src/App.tsx', 'react', 'tsx'],
    ['App.jsx', 'react', 'jsx'],
    ['src/app.component.ts', 'angular', 'typescript'],
    ['angular.json', 'angular', 'json'],
    ['App.svelte', 'svelte', 'svelte'],
    ['Dockerfile', 'docker', 'dockerfile'],
    ['compose.yaml', 'docker', 'yaml'],
    ['src/main.rs', 'rust', 'rust'],
    ['source.py', 'python', 'python'],
    ['MAIN.CPP', 'cpp', 'cpp'],
    ['src/java/name.txt', undefined, 'txt'],
  ] as const) {
    expect(fileType(path)).toBe(type);
    expect(codeLanguage(path)).toBe(syntax);
  }
});
