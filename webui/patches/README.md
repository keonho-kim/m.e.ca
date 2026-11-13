# Runtime patches

Bun applies the exact versions in `package.json` → `patchedDependencies` during frozen installs, including installs with lifecycle scripts disabled. The application adapters and tests own the consumer contracts; the patches cover the required package behavior.

## Code editor

`react-simple-code-editor@0.14.1.patch` selects the published TSX as its ESM entry. It admits text before recording changes, groups IME composition into one undo operation, exposes an explicit history-group boundary and provides a fixed-height native textarea without a duplicate `pre` for large documents. Typing and Enter inspect the current line; bulk beforeinput and paste use one selection replacement. Undo metadata is capped at 100 records.

The application owns UTF-8 admission, undo-text budgets and deferred Astryx highlighting. Preview reads leave undo history intact. Constant CSS is a style text child and source highlighting is React text; edited document content does not enter an HTML sink.

Contracts: `e2e/editor.spec.ts`, `e2e/editor-boundaries.spec.ts` and `e2e/browser-policy.spec.ts`.

## Streaming Markdown

`@astryxdesign%2Fcore@0.7.14.patch` separates Astryx incremental parsing (`isStreaming`) from character reveal (`isRevealing`). The animation frame stops when text catches up. Resuming reveal after an instant catch-up or a reduced-motion change preserves the displayed prefix. Source, published runtime and declarations carry the same contract.

The application uses Astryx's parser and AST transform plugin. Covered-content projection belongs to the workspace presentation model; a second parser or an unsupported AST-input prop is not part of this adapter.

Contracts: `e2e/performance.spec.ts` and `e2e/presentation-budget.spec.ts`.

## Maintenance boundaries

Review and rebase each patch when changing its pinned package, then run the corresponding browser contracts. Preserve upstream licenses and include package source, runtime and declaration changes required by the published entry points.

These are diffs against third-party files. Repository authored-file size and header rules apply to the adapters and tests; upstream files retain their package structure. Unified-diff blank context lines require a leading space, which `.gitattributes` preserves during Git whitespace checks.
