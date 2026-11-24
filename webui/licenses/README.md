# Browser license inputs

These files supply the version-specific fallbacks selected by
[`scripts/notices.ts`](../scripts/notices.ts) when a locked npm package omits its
root license. Only full embeds the browser bundle and distributes its notices.

- Astryx Core and Neutral `0.7.17` use the [local MIT text](astryx-0.7.17.LICENSE) from https://github.com/meta/astryx/blob/v0.7.17/LICENSE. Both npm packages declare MIT and omit their root license text. The fallbacks match their exact versions.
- StyleX `0.19.1`: [local license text](stylex-0.19.1.LICENSE), from https://github.com/facebook/stylex/blob/0.19.1/LICENSE.

The fallback map matches only these exact package versions. A new omission fails
the build until its source and license are reviewed. Other bundled package
notices come from the locked npm distributions; the stylesheet-only Tailwind
package is included explicitly. Build output is packaged as
`WEBUI_THIRD_PARTY_NOTICES.txt` in full archives and is available at
`/ui/THIRD_PARTY_NOTICES.txt` when the embedded UI runs.
