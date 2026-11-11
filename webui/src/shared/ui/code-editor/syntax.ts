// Purpose: Paint bounded editor syntax through Astryx's public tokenizer and CSS Highlight API.
// Pattern: Cancellable presentation adapter without token DOM nodes or HTML injection.
// Usage: The editor owns cancellation and removes painted ranges on edits, preview and unmount.
// Related: webui/src/shared/ui/code-editor/index.tsx, webui/src/shared/lib/code-language/index.ts.
import { applyHighlightRangesFlat, tokenizeStreaming } from '@astryxdesign/core/CodeBlock';

export const syntaxClasses = [
  '[&::highlight(astryx-keyword)]:text-(--color-syntax-keyword)',
  '[&::highlight(astryx-string)]:text-(--color-syntax-string)',
  '[&::highlight(astryx-comment)]:text-(--color-syntax-comment)',
  '[&::highlight(astryx-number)]:text-(--color-syntax-number)',
  '[&::highlight(astryx-function)]:text-(--color-syntax-function)',
  '[&::highlight(astryx-type)]:text-(--color-syntax-type)',
  '[&::highlight(astryx-tag)]:text-(--color-syntax-tag)',
  '[&::highlight(astryx-attribute)]:text-(--color-syntax-attribute)',
  '[&::highlight(astryx-property)]:text-(--color-syntax-property)',
  '[&::highlight(astryx-operator)]:text-(--color-syntax-operator)',
  '[&::highlight(astryx-constant)]:text-(--color-syntax-constant)',
  '[&::highlight(astryx-punctuation)]:text-(--color-syntax-punctuation)',
  '[&::highlight(astryx-variable)]:text-(--color-syntax-variable)',
].join(' ');

export function canHighlight(text: string) {
  if (text.length > 65536) return false;
  let lines = 1,
    length = 0;
  for (let i = 0; i < text.length; i++) {
    if (text.charCodeAt(i) === 10) {
      if (++lines > 2000) return false;
      length = 0;
    } else if (++length > 8192) return false;
  }
  return true;
}

export async function syntaxTokens(text: string, language: string, signal: AbortSignal) {
  if (!canHighlight(text)) return;
  signal.throwIfAborted();
  const lines: Parameters<typeof applyHighlightRangesFlat>[1] = [];
  const budget = new AbortController();
  let count = 0;
  await tokenizeStreaming(
    text,
    language,
    (batch) => {
      signal.throwIfAborted();
      for (const line of batch) {
        count += line.length;
        if (count > 4096) {
          budget.abort();
          return;
        }
        lines.push(line);
      }
    },
    AbortSignal.any([signal, budget.signal])
  );
  signal.throwIfAborted();
  if (!budget.signal.aborted) return lines;
}

export async function paintSyntax(
  element: HTMLElement,
  text: string,
  language: string,
  signal: AbortSignal
) {
  if (typeof CSS === 'undefined' || !('highlights' in CSS) || typeof Highlight === 'undefined')
    return;
  const lines = await syntaxTokens(text, language, signal);
  if (lines?.length) return applyHighlightRangesFlat(element, lines);
}
