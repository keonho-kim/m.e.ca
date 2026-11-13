// Purpose: Include upstream browser runtime license texts in the distributed embedded application.
// Pattern: Vite build-output adapter.
// Usage: vite.config registers runtimeNotices; package.ts embeds its emitted text asset.
// Related: THIRD_PARTY_NOTICES.md, webui/package.json.
import { readdir, readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import type { Plugin } from 'vite';

export function runtimeNotices(): Plugin {
  return {
    name: 'meca-runtime-notices',
    async generateBundle(_options, bundle) {
      const roots = new Set<string>(),
        packages = new Map<
          string,
          {
            name: string;
            version: string;
            license?: string;
            repository?: string | { url?: string };
          }
        >();
      for (const asset of Object.values(bundle)) {
        if (asset.type !== 'chunk') continue;
        for (const module of Object.keys(asset.modules)) {
          if (!module.includes('/node_modules/')) continue;
          const marker = '/node_modules/',
            at = module.lastIndexOf(marker);
          const prefix = module.slice(0, at + marker.length),
            parts = module.slice(at + marker.length).split('/');
          const path = join(prefix, ...parts.slice(0, parts[0]?.startsWith('@') ? 2 : 1));
          if (roots.has(path)) continue;
          roots.add(path);
          const pkg = JSON.parse(await readFile(join(path, 'package.json'), 'utf8'));
          packages.set(path, pkg);
        }
      }
      const fallback: Record<string, string> = {
        '@astryxdesign/core@0.7.14': 'astryx-core-0.7.14.LICENSE',
        '@stylexjs/stylex@0.19.1': 'stylex-0.19.1.LICENSE',
      };
      const missing: string[] = [];
      const cssRoot = fileURLToPath(new URL('../node_modules/tailwindcss/', import.meta.url));
      packages.set(cssRoot, JSON.parse(await readFile(join(cssRoot, 'package.json'), 'utf8')));
      // Material glyphs are compiled to local SVG; retain their upstream notice.
      const iconRoot = fileURLToPath(
        new URL('../node_modules/@mui/icons-material/', import.meta.url)
      );
      packages.set(iconRoot, JSON.parse(await readFile(join(iconRoot, 'package.json'), 'utf8')));
      const sections = ['MECA embedded WebUI — third-party runtime notices\n'];
      for (const [path, pkg] of [...packages].sort((a, b) => a[1].name.localeCompare(b[1].name))) {
        const texts = (await readdir(path)).filter((name) =>
          /^(?:licen[sc]e|copying|notice)(?:\.|$)/i.test(name)
        );
        const repository =
          typeof pkg.repository === 'string' ? pkg.repository : pkg.repository?.url;
        sections.push(
          `\n${pkg.name} ${pkg.version}\nLicense: ${pkg.license ?? 'See upstream license'}\n${repository ?? ''}\n`
        );
        if (!texts.length) {
          const file = fallback[`${pkg.name}@${pkg.version}`];
          if (file)
            sections.push(
              await readFile(fileURLToPath(new URL(`../licenses/${file}`, import.meta.url)), 'utf8')
            );
          else missing.push(`${pkg.name}@${pkg.version}`);
        }
        for (const name of texts.sort())
          sections.push(`\n${name}\n${await readFile(join(path, name), 'utf8')}\n`);
      }
      if (missing.length) throw Error(`Missing distributed license texts: ${missing.join(', ')}`);
      this.emitFile({
        type: 'asset',
        fileName: 'THIRD_PARTY_NOTICES.txt',
        source: sections.join('\n'),
      });
    },
  };
}
