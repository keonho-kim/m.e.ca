# Editor runtime patch

`react-simple-code-editor@0.14.1.patch` is applied by Bun's pinned `patchedDependencies` during frozen installs, including installs with lifecycle scripts disabled. It keeps the upstream MIT license and published editor rather than maintaining a second editor implementation.

The patch selects the package's published TSX as its ESM entry, admits text before recording changes, groups IME composition into one undo operation, exposes an explicit history-group boundary and uses a fixed-height native textarea without a duplicate `pre` for large documents. Normal typing and Enter find the current line without splitting the complete prefix. Bulk native beforeinput and paste use one explicit selection replacement to avoid per-line browser insertion work. Undo metadata remains capped at 100 records.

The application owns UTF-8 admission, history text budgets and deferred Astryx highlighting. Preview reads do not modify history. Browser contracts in `e2e/editor.spec.ts` and `e2e/editor-boundaries.spec.ts` verify the patched runtime. Update this patch deliberately when changing the pinned package; a dependency upgrade must not silently discard these contracts.

The patch is a diff against third-party source and generated declarations. The upstream source remains intact as a third-party file; repository authored-file size/header rules apply to the adapter and tests rather than requiring a vendored rewrite of the upstream component.

Unified-diff blank context lines contain a required leading space. `.gitattributes` preserves those markers during Git whitespace checks; authored source keeps its normal whitespace rules.
