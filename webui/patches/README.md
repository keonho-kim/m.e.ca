# Runtime patches

Bun applies the exact versions in `package.json` → `patchedDependencies` during frozen installs, including installs with lifecycle scripts disabled. The application adapters and tests own the consumer contracts; the patches cover the required package behavior.

## Code editor

`react-simple-code-editor@0.14.1.patch` selects the published TSX as its ESM entry. It admits text before recording changes, groups IME composition into one undo operation, exposes an explicit history-group boundary and provides a fixed-height native textarea without a duplicate `pre` for large documents. Typing and Enter inspect the current line; bulk beforeinput and paste use one selection replacement. Undo metadata is capped at 100 records.

The application owns UTF-8 admission, undo-text budgets and deferred Astryx highlighting. Preview reads leave undo history intact. Constant CSS is a style text child and source highlighting is React text; edited document content does not enter an HTML sink.

Contracts: `e2e/editor.spec.ts`, `e2e/editor-boundaries.spec.ts` and `e2e/browser-policy.spec.ts`.

## Streaming Markdown

`@astryxdesign%2Fcore@0.7.17.patch` separates Astryx incremental parsing (`isStreaming`) from character reveal (`isRevealing`). The animation frame stops when text catches up. Resuming reveal after an instant catch-up or a reduced-motion change preserves the displayed prefix. Source, published runtime and declarations carry the same contract.

The application uses Astryx's parser and AST transform plugin. Covered-content projection belongs to the workspace presentation model; a second parser or an unsupported AST-input prop is not part of this adapter.

Contracts: `e2e/performance.spec.ts` and `e2e/presentation-budget.spec.ts`.

## Chat scroll ownership

The Astryx patch keeps the jump button hidden while automatic following is locked. Stream growth, virtual-history measurement and programmatic position changes do not imply reader intent. Upward wheel/keyboard input, touch movement and scrollbar dragging can release the lock; nested scrolling keeps its own input. Focusing transcript controls pauses following. An outstanding upward-reading intent prevents a late scrollend from re-locking before the user scrolls back down. The public layout context exposes the imperative follow state and navigation actions so the virtualizer disables competing corrections while following. Returning to the bottom clears the affordance, and unmount cancels pending animation frames and input listeners.

Contract: `e2e/chat-stream-order.spec.ts`, including normal motion and history handoff in Chromium, Firefox and WebKit.

## Maintenance boundaries

Core, Neutral and CLI are pinned together at `0.7.17`. The `0.7.16` to `0.7.17` CLI upgrade completed with no application codemods or manual migrations. The streaming and scroll contracts above still need this patch. Its Markdown merge preserves the upstream list bounds, table layout, link handling and prop diagnostics. Release fixes for selector focus, narrow composers, input keyboard avoidance, clipboard fallback and theme contrast are supplied by the upgraded packages. [Upstream release](https://github.com/meta/astryx/releases/tag/v0.7.17)

Review and rebase each patch when changing its pinned package, then run the corresponding browser contracts. Preserve upstream licenses and include package source, runtime and declaration changes required by the published entry points.

These are diffs against third-party files. Repository authored-file size and header rules apply to the adapters and tests; upstream files retain their package structure. Unified-diff blank context lines require a leading space, which `.gitattributes` preserves during Git whitespace checks.
