// Purpose: Compile selected MUI Material glyphs into dependency-free SVG components.
// Pattern: Build-time asset adapter with a pinned upstream license.
// Usage: The check/build scripts generate the file icon module before TypeScript compilation.
// Related: webui/src/shared/ui/file-icon/index.tsx, webui/scripts/notices.ts.
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

const names = {
  code: 'CodeOutlined',
  javascript: 'JavascriptOutlined',
  html: 'HtmlOutlined',
  css: 'CssOutlined',
  framework: 'IntegrationInstructionsOutlined',
  coffee: 'CoffeeOutlined',
  data: 'DataObjectOutlined',
  document: 'ArticleOutlined',
  php: 'PhpOutlined',
  schema: 'SchemaOutlined',
  database: 'StorageOutlined',
  terminal: 'TerminalOutlined',
  file: 'DescriptionOutlined',
};
const target = fileURLToPath(new URL('../src/shared/ui/file-icon/generated.tsx', import.meta.url));
const entries: string[] = [];
for (const [name, source] of Object.entries(names)) {
  const module = await import(`@mui/icons-material/${source}`);
  const markup = renderToStaticMarkup(createElement(module.default));
  const svg = markup.match(/<svg\b[^>]*viewBox="([^"]+)"[^>]*>([\s\S]*?)<\/svg>/);
  if (!svg || /<(?!\/?(?:path|g|circle|rect|polygon|polyline|line|ellipse)\b)/.test(svg[2] ?? ''))
    throw Error(`Unsupported SVG shape: ${source}`);
  const content = svg[2]
    ?.replaceAll('fill-rule=', 'fillRule=')
    .replaceAll('clip-rule=', 'clipRule=');
  entries.push(
    `${JSON.stringify(name)}: (props: SVGProps<SVGSVGElement>) => <svg {...props} viewBox=${JSON.stringify(svg[1])} fill="currentColor">${content}</svg>`
  );
}
const output = `// Purpose: Preserve selected upstream Material glyphs without their styling runtime.
// Pattern: Generated static icon dictionary; do not edit.
// Usage: scripts/file-icons.ts generates this module for the Astryx file icon adapter.
// Related: webui/scripts/file-icons.ts, webui/src/shared/ui/file-icon/index.tsx.
// Source: @mui/icons-material; its MIT license is included in distributed browser notices.
import type { SVGProps } from 'react';
export const fileGlyphs = {\n${entries.join(',\n')}\n};\n`;
await mkdir(fileURLToPath(new URL('../src/shared/ui/file-icon/', import.meta.url)), {
  recursive: true,
});
if ((await readFile(target, 'utf8').catch(() => '')) !== output) await writeFile(target, output);
