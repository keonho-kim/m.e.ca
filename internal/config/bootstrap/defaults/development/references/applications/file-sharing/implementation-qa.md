# File sharing implementation Q&A

## When can an upload be attached to a record?

Attach a verified finished resource when the feature requires a ready file. If early visible association is useful, attach the scoped pending identity and show its phase. Check that the upload belongs to the same actor and parent, and promote it only when the service acknowledges the required outcome.

## What can be retried after partial success?

Keep each persisted successful file and the associated record. Retry the transfer or processing of the failed item with the original attempt identity where supported. An uncertain association is resolved before adding another one. Show which steps are still pending across refresh.

## How can a shared link be revoked?

Choose between per-request authorization or a bounded capability lifetime according to the actual revocation guarantee. A directly usable capability may remain usable until expiration. If the product requires immediate removal, enforce current scope at a controlled delivery boundary and verify older links and previews.

## How are replacements and deletions managed?

Decide whether replacement changes a version of the same resource or creates a new file identity. Preserve links to historical versions only for the eligible audience. Before deleting stored content, resolve references, retention and the requested removal scope, including derived previews.
