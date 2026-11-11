// Purpose: Render model and file Markdown with Astryx while keeping remote media explicit.
// Pattern: Restricted content renderer.
// Usage: Conversation and skill/file reading share this component.
// Related: webui/src/pages/workspace/ui/chat/chat.tsx, webui/src/pages/workspace/ui/skills/skills.tsx.

import { Button } from '@astryxdesign/core/Button';
import { useMediaQuery } from '@astryxdesign/core/hooks';
import { Link } from '@astryxdesign/core/Link';
import { Markdown } from '@astryxdesign/core/Markdown';
import { Text } from '@astryxdesign/core/Text';
import { createContext, memo, useContext, useEffect, useMemo, useState } from 'react';
import { useLocale } from '@/shared/i18n';
import { CodePreview } from '@/shared/ui/code-preview';
import { markdownBudget } from '@/shared/ui/markdown/pagination';

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

function useRevealDuringBurst(text: string, enabled: boolean) {
  const [settledText, setSettledText] = useState(enabled ? '' : text);
  const active = enabled && !!text && settledText !== text;
  useEffect(() => {
    if (!active) return;
    // Astryx's reveal requests frames even after its text catches up. A quiet
    // burst settles to the received text and releases the animation clock.
    const timer = setTimeout(() => {
      setSettledText(text);
    }, 500);
    return () => clearTimeout(timer);
  }, [active, text]);
  return active;
}

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
  const [tail] = useState(streaming);
  const [page, setPage] = useState(tail ? Number.MAX_SAFE_INTEGER : 0);
  const follow = tail && page === Number.MAX_SAFE_INTEGER;
  const reducedMotion = useMediaQuery('(prefers-reduced-motion: reduce)');
  const reveal = useRevealDuringBurst(text, streaming && follow && !reducedMotion);
  const plugins = useMemo(() => [markdownBudget(page, setPage)], [page]);
  const renderers = useMemo(
    () => ({
      ...components,
      link: ({ href, children }: { href: string; children: React.ReactNode }) => {
        if (onReference && !/^(?:[a-z]+:|\/\/|#)/i.test(href))
          return (
            <Button
              type="button"
              className="reference-link max-w-full h-auto"
              onClick={() => onReference(href)}
              label={String(children)}
              variant="ghost"
            >
              <Text className="whitespace-normal break-words text-left">{children}</Text>
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
    <FollowCode value={follow}>
      <Markdown
        className="meca-prose min-w-0 max-w-full [overflow-wrap:anywhere]"
        isStreaming={streaming && follow && (reveal || reducedMotion)}
        components={renderers}
        plugins={plugins}
        contentWidth="100%"
        data-tone={tone}
      >
        {text}
      </Markdown>
    </FollowCode>
  );
});
