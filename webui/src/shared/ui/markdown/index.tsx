// Purpose: Render model and file Markdown with Astryx while keeping remote media explicit.
// Pattern: Restricted content renderer.
// Usage: Conversation and skill/file reading share this component.
// Related: webui/src/pages/workspace/ui/chat/chat.tsx, webui/src/pages/workspace/ui/skills/skills.tsx.

import { Button } from '@astryxdesign/core/Button';
import { useMediaQuery } from '@astryxdesign/core/hooks';
import { Link } from '@astryxdesign/core/Link';
import { Markdown } from '@astryxdesign/core/Markdown';
import { Popover } from '@astryxdesign/core/Popover';
import { Text } from '@astryxdesign/core/Text';
import { createContext, memo, useContext, useEffect, useMemo, useRef, useState } from 'react';
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

const CompleteText = memo(function CompleteText({ text }: { text: string }) {
  const t = useLocale();
  const [open, setOpen] = useState(false);
  return (
    <Popover
      label={t('textView')}
      placement="above"
      width={720}
      isOpen={open}
      onOpenChange={setOpen}
      content={open ? <CodePreview code={text} /> : null}
    >
      <Button label={t('textView')} size="sm" variant="ghost" />
    </Popover>
  );
});
export const MarkdownText = memo(function MarkdownText({
  text,
  streaming = false,
  showCompleteText = true,
  tone,
  onReference,
}: {
  text: string;
  streaming?: boolean;
  showCompleteText?: boolean;
  tone?: 'thinking';
  onReference?: (href: string) => void;
}) {
  const tail = useRef(streaming);
  const reducedMotion = useMediaQuery('(prefers-reduced-motion: reduce)');
  const reveal = useRevealDuringBurst(text, streaming && tail.current && !reducedMotion);
  const plugins = useMemo(() => [markdownBudget(tail.current)], []);
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
    <>
      <FollowCode value={tail.current}>
        <Markdown
          className="meca-prose min-w-0 max-w-full [overflow-wrap:anywhere]"
          isStreaming={streaming && (reveal || reducedMotion)}
          components={renderers}
          plugins={plugins}
          contentWidth="100%"
          data-tone={tone}
        >
          {text}
        </Markdown>
      </FollowCode>
      {showCompleteText && hasMarkdownWindow(text) && <CompleteText text={text} />}
    </>
  );
});
