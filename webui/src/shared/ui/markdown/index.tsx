// Purpose: Render model and file Markdown with Astryx while keeping remote media explicit.
// Pattern: Restricted content renderer.
// Usage: Conversation and skill/file reading share this component.
// Related: webui/src/pages/workspace/ui/chat/chat.tsx, webui/src/pages/workspace/ui/skills/skills.tsx.

import { Button } from '@astryxdesign/core/Button';
import { Link } from '@astryxdesign/core/Link';
import { Markdown } from '@astryxdesign/core/Markdown';
import { Popover } from '@astryxdesign/core/Popover';
import { createContext, memo, useContext, useMemo, useRef } from 'react';
import { useLocale } from '@/shared/i18n';
import { CodePreview } from '@/shared/ui/code-preview';
import { hasMarkdownWindow, markdownBudget } from '@/shared/ui/markdown/render-budget';

function safeLink(href: string) {
  try {
    const url = new URL(href, window.location.href);
    return ['https:', 'http:', 'mailto:'].includes(url.protocol) ? href : undefined;
  } catch {
    return undefined;
  }
}
function ExternalImage({ src, alt }: { src: string; alt: string }) {
  const t = useLocale();
  return (
    <Link
      href={safeLink(src)}
      target="_blank"
      rel="noopener noreferrer"
      className="external-image-link"
    >
      {alt || t('remoteImage')}
    </Link>
  );
}
const FollowCode = createContext(false);
function MarkdownCode({ code, language }: { code: string; language?: string }) {
  return <CodePreview code={code} language={language} follow={useContext(FollowCode)} />;
}
const components = {
  code: MarkdownCode,
  image: ExternalImage,
  link: ({ href, children }: { href: string; children: React.ReactNode }) => (
    <Link href={safeLink(href)} target="_blank" rel="noopener noreferrer">
      {children}
    </Link>
  ),
};
export const MarkdownText = memo(function MarkdownText({
  text,
  streaming = false,
  tone,
  onReference,
}: {
  text: string;
  streaming?: boolean;
  tone?: 'thinking';
  onReference?: (href: string) => void;
}) {
  const t = useLocale();
  const tail = useRef(streaming);
  const plugins = useMemo(() => [markdownBudget(tail.current)], []);
  const renderers = useMemo(
    () => ({
      ...components,
      link: ({ href, children }: { href: string; children: React.ReactNode }) => {
        if (onReference && !/^(?:[a-z]+:|\/\/|#)/i.test(href))
          return (
            <Button
              type="button"
              className="reference-link"
              onClick={() => onReference(href)}
              label={String(children)}
              variant="ghost"
            >
              {children}
            </Button>
          );
        return (
          <Link href={safeLink(href)} target="_blank" rel="noopener noreferrer">
            {children}
          </Link>
        );
      },
    }),
    [onReference]
  );
  return (
    <>
      <FollowCode value={tail.current}>
        <Markdown
          isStreaming={streaming}
          components={renderers}
          plugins={plugins}
          contentWidth="100%"
          data-tone={tone}
        >
          {text}
        </Markdown>
      </FollowCode>
      {hasMarkdownWindow(text) && (
        <Popover
          label={t('textView')}
          placement="above"
          width={720}
          content={<CodePreview code={text} />}
        >
          <Button label={t('textView')} size="sm" variant="ghost" />
        </Popover>
      )}
    </>
  );
});
