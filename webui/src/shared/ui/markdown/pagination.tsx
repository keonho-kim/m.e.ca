// Purpose: Add inline page controls to large Markdown without a second parse or a full-text modal.
// Pattern: Astryx public AST transform and extension renderer.
// Usage: MarkdownText owns the selected page and supplies its setter to this plugin.
// Related: webui/src/shared/ui/markdown/render-budget.ts, webui/src/shared/ui/markdown/index.tsx.
import { Button } from '@astryxdesign/core/Button';
import { HStack } from '@astryxdesign/core/Layout';
import {
  createMarkdownPlugin,
  type MarkdownExtensionNode,
} from '@astryxdesign/core/Markdown/plugins';
import { Text } from '@astryxdesign/core/Text';
import { useLocale } from '@/shared/i18n';
import { markdownWindow } from '@/shared/ui/markdown/render-budget';

type Pager = MarkdownExtensionNode<
  'meca-render-pages',
  'pages',
  { index: number; pages: number },
  'block'
>;
function PageControls({
  index,
  pages,
  onPage,
}: {
  index: number;
  pages: number;
  onPage: (index: number) => void;
}) {
  const t = useLocale();
  return (
    <HStack
      gap={3}
      align="center"
      wrap="wrap"
      className="markdown-pages"
      role="group"
      aria-label={t('contentPages')}
    >
      <Button
        size="sm"
        variant="ghost"
        label={t('pagePrevious')}
        isDisabled={index === 0}
        onClick={() => onPage(index - 1)}
      />
      <Text type="supporting" color="secondary">
        {t('contentPages')} {index + 1} / {pages}
      </Text>
      <Button
        size="sm"
        variant="ghost"
        label={t('pageNext')}
        isDisabled={index === pages - 1}
        onClick={() => onPage(index + 1 === pages - 1 ? Number.MAX_SAFE_INTEGER : index + 1)}
      />
    </HStack>
  );
}
export function markdownBudget(page: number, onPage: (index: number) => void) {
  return createMarkdownPlugin<'meca-render-pages', Pager>({
    name: 'meca-render-pages',
    apiVersion: 1,
    transform(document, context) {
      if (context.display !== 'block') return document;
      const window = markdownWindow(document, page);
      if (window.pages === 1) return document;
      const pager: Pager = {
        type: 'extension',
        plugin: 'meca-render-pages',
        name: 'pages',
        display: 'block',
        data: { index: window.index, pages: window.pages },
      };
      return { ...window.document, children: [...window.document.children, pager] };
    },
    renderers: {
      pages: {
        render: ({ node }) => <PageControls {...node.data} onPage={onPage} />,
        toText: () => '',
      },
    },
  });
}
