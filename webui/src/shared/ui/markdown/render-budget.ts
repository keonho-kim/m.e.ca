// Purpose: Bound rendered Markdown nodes through Astryx's immutable AST plugin contract.
// Pattern: Bounded presentation projection preserving source syntax and node provenance.
// Usage: MarkdownText uses this after Astryx's existing incremental parse.
// Related: webui/src/shared/ui/markdown/index.tsx.
import { createMarkdownPlugin } from '@astryxdesign/core/Markdown/plugins';

type Branch = { readonly type: string; readonly children?: readonly Branch[] };
export function markdownBudget(tail: boolean) {
  return createMarkdownPlugin({
    name: 'meca-render-window',
    apiVersion: 1,
    transform(document, context) {
      if (context.display !== 'block') return document;
      if (!hasMarkdownWindow(context.source)) return document;
      let remaining = 1024;
      function bound<T extends Branch>(node: T): T | undefined {
        if (remaining-- <= 0) return undefined;
        if (!node.children) return node;
        const result: Branch[] = [],
          children = node.children;
        for (let index = 0; index < children.length && remaining > 0; index++) {
          const child = children[tail ? children.length - 1 - index : index];
          if (!child) continue;
          const value = bound(child);
          if (value) result.push(value);
        }
        if (tail) result.reverse();
        if (
          result.length === children.length &&
          result.every((child, index) => child === children[index])
        )
          return node;
        // Filtering children preserves the concrete node's content kinds and provenance.
        return { ...node, children: result };
      }
      return bound(document) ?? { ...document, children: [] };
    },
  });
}
export function hasMarkdownWindow(text: string) {
  if (text.length > 1024) return true;
  let lines = 0;
  for (const char of text) if (char === '\n' && ++lines > 48) return true;
  return false;
}
