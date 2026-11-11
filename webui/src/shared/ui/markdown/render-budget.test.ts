// Purpose: Keep every Markdown leaf accessible across bounded pages without reparsing its source.
// Pattern: Pure AST projection contracts over Astryx's public parser.
// Usage: bun test src/shared/ui/markdown/render-budget.test.ts.
// Related: webui/src/shared/ui/markdown/render-budget.ts.
import { expect, test } from 'bun:test';
import { parseMarkdownAst } from '@astryxdesign/core/Markdown';
import { markdownWindow } from '@/shared/ui/markdown/render-budget';

type Branch = { type: string; value?: string; children?: readonly Branch[] };
function leaves(node: Branch): string[] {
  return node.children?.flatMap(leaves) ?? (node.value ? [node.value] : []);
}
function nodes(node: Branch): number {
  return 1 + (node.children?.reduce((sum, child) => sum + nodes(child), 0) ?? 0);
}
test('dense nested Markdown pages preserve all text leaves and retain the source tree', () => {
  const document = parseMarkdownAst(
    Array.from({ length: 1800 }, (_, index) => `- Entry **${index}**`).join('\n')
  );
  const before = JSON.stringify(document),
    first = markdownWindow(document, 0);
  const actual: string[] = [];
  expect(first.pages).toBeGreaterThan(1);
  for (let index = 0; index < first.pages; index++) {
    const page = markdownWindow(document, index);
    actual.push(...leaves(page.document));
    expect(nodes(page.document)).toBeLessThanOrEqual(1024);
  }
  expect(actual).toEqual(leaves(document));
  expect(JSON.stringify(document)).toBe(before);
  expect(markdownWindow(document, Number.MAX_SAFE_INTEGER).index).toBe(first.pages - 1);
});
test('ordered list continuations retain their marker and table pages retain a bounded header', () => {
  const document = parseMarkdownAst(
    Array.from({ length: 1200 }, (_, index) => `${index + 7}. Entry ${index}`).join('\n')
  );
  const page = markdownWindow(document, 1).document.children[0];
  expect(page?.type).toBe('list');
  if (page?.type === 'list') {
    expect(page.start).toBeGreaterThan(7);
    expect(page.position).toBeUndefined();
  }
  const table = parseMarkdownAst(
    '| Name | Value |\n|---|---|\n' +
      Array.from({ length: 600 }, (_, index) => `| Row ${index} | ${index} |`).join('\n')
  );
  const part = markdownWindow(table, 1).document;
  expect(nodes(part)).toBeLessThanOrEqual(1024);
  expect(leaves(part).slice(0, 2)).toEqual(['Name', 'Value']);
  const small = parseMarkdownAst('Small **complete** response.');
  expect(markdownWindow(small, 0).document).toBe(small);
});
