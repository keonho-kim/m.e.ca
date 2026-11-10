# Browser license inputs

The npm distributions below omit their root license text. These exact upstream
release files are retained for reproducible offline license packaging:

- Astryx Core 0.7.11: https://github.com/meta/astryx/blob/v0.7.11/LICENSE
- StyleX 0.19.1: https://github.com/facebook/stylex/blob/0.19.1/LICENSE

`../scripts/notices.ts` maps only these exact package versions. New omissions
fail the build until their matching upstream notices are reviewed. License texts
for other bundled packages come directly from the locked npm distributions.
