# Browser license inputs

These files supply the version-specific fallbacks selected by
[`scripts/notices.ts`](../scripts/notices.ts) when a locked npm package omits its
root license. Only full embeds the browser bundle and distributes its notices.

- Astryx Core `0.7.12-canary.f124c2f`: [local MIT text](astryx-core-0.7.12-canary.f124c2f.LICENSE), retrieved from https://github.com/meta/astryx/blob/main/LICENSE. The package declares MIT; this fallback is not a commit-specific license snapshot.
- StyleX `0.19.1`: [local license text](stylex-0.19.1.LICENSE), from https://github.com/facebook/stylex/blob/0.19.1/LICENSE.

The fallback map matches only these exact package versions. A new omission fails
the build until its source and license are reviewed. Other bundled package
notices come from the locked npm distributions; the stylesheet-only Tailwind
package is included explicitly. Build output is packaged as
`WEBUI_THIRD_PARTY_NOTICES.txt` in full archives and is available at
`/ui/THIRD_PARTY_NOTICES.txt` when the embedded UI runs.
