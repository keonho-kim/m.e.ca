# Collaborative documents and wikis

Use this when people edit a shared document, discuss changes and recover prior content. Choose the ownership and revision model before adding autosave or simultaneous editing.

[Do & Don’t](do-and-dont.md) · [Implementation Q&A](implementation-qa.md)

## Main flow

1. Open a document under the current view or edit permission. Identify the revision and any local draft. Give the document a stable route.
2. Apply edits using the expected editor model. Show unsaved, saving, saved and conflict states as applicable. Retain recoverable work if the write fails.
3. Add comments or suggestions where required. Bind them to the document and an applicable position or revision. Explain when changed content makes an anchor obsolete.
4. Open authorized history and compare or restore a revision. Record a new current revision when the product preserves history.

## Implementation boundaries

- Select whole-document writes with a base revision, structured patches or a concurrent editing model from the actual collaboration requirement.
- Treat document revisions as authoritative. Keep presence, selection indicators and draft previews ephemeral or scoped as appropriate.
- Verify current permissions for editing, history, comments, preview and export. Decide how a disconnected editor handles local drafts after access is revoked.
- Keep revision and comment lifetimes explicit. Preserve recoverability while honoring the product's deletion and visibility policy.
- In a canvas-style editor, store shared object size and position in document coordinates and bind edits to the relevant revision. Apply the same edit contract to drag, placement controls and precise position fields.

## Borrow for this workflow

Use [Content publishing](../content-publishing/app-guide.md) for public release and draft preview. Use [File sharing](../file-sharing/app-guide.md) for linked files and capability expiry. Use [Offline fieldwork](../offline-fieldwork/app-guide.md) if disconnection must preserve editable work.

Google Docs Editors Help “Find out what’s changed in a file” describes version history and restoration.

## Acceptance cases

- Two editors save from the same base. The product merges, accepts or flags the conflict according to the chosen contract without silently losing either person's recoverable work.
- Restore an older revision and verify history and comments according to the restore policy.
- A user's access is removed while their editor is open. A later save and export follow current permissions.
