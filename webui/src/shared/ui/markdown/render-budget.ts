// Purpose: Page immutable Markdown nodes without dropping source content or losing container context.
// Pattern: Pure bounded AST projection over Astryx's existing parse.
// Usage: The pagination plugin renders one window and native inline page controls.
// Related: webui/src/shared/ui/markdown/pagination.tsx, webui/src/shared/ui/markdown/render-budget.test.ts.
type Branch = {
  readonly type: string;
  readonly children?: readonly Branch[];
  readonly ordered?: boolean;
  readonly start?: number;
};
// Leave room for repeated ancestor paths, a small table header and page controls.
const pageNodes = 768;
const sizes = new WeakMap<Branch, number>();
function count(node: Branch): number {
  if (!node.children?.length) return 1;
  const known = sizes.get(node);
  if (known !== undefined) return known;
  const size = 1 + node.children.reduce((sum, child) => sum + count(child), 0);
  sizes.set(node, size);
  return size;
}
export function markdownWindow<T extends Branch>(document: T, requested: number) {
  const pages = Math.ceil(Math.max(1, count(document) - 1) / pageNodes);
  const index = Math.min(Math.max(0, requested), pages - 1);
  if (pages === 1) return { document, pages, index };
  const start = 1 + index * pageNodes,
    end = start + pageNodes;
  let cursor = 0;
  function select<N extends Branch>(node: N): N | undefined {
    if (cursor >= end) return undefined;
    const size = count(node);
    if (cursor + size <= start) {
      cursor += size;
      return undefined;
    }
    if (cursor >= start && cursor + size <= end) {
      cursor += size;
      return node;
    }
    const position = cursor++;
    if (!node.children) return position >= start ? node : undefined;
    const children: Branch[] = [];
    let first = -1;
    for (let child = 0; child < node.children.length && cursor < end; child++) {
      const source = node.children[child];
      if (!source) continue;
      const selected = select(source);
      if (selected) {
        if (first < 0) first = child;
        children.push(selected);
      }
    }
    if (!children.length) return undefined;
    if (node.type === 'table' && first > 0) {
      const header = node.children[0];
      if (header && count(header) <= 128) children.unshift(header);
    }
    return {
      ...node,
      children,
      ...(node.type === 'list' && node.ordered && first > 0
        ? // The continuation container is synthetic; its children retain source provenance.
          { start: (node.start ?? 1) + first, position: undefined }
        : {}),
    };
  }
  return { document: select(document) ?? { ...document, children: [] }, pages, index };
}
