# File upload, storage and sharing

Use this when a file becomes a durable resource or an attachment that people can preview, download or share. Start from the parent record's audience and the actual conditions for an available file.

[Do & Don’t](do-and-dont.md) · [Implementation Q&A](implementation-qa.md)

## Main flow

1. Choose or drop local files into the intended destination and display the limits relevant to that use. Use the same scoped upload attempt and per-file identity for either input.
2. Transfer and finalize it. Verify the completed resource, required checks and ownership. Show transfer, processing and ready states with their actual outcomes.
3. Associate the ready file with a parent record or collection. Generate previews where needed and use their own pending or failed states.
4. Access, share, replace or delete the file according to current permissions. Keep useful recovery and per-file results for partial failures.

## Implementation boundaries

- Keep file identity, version, storage location and audience explicit. Give an upload session and ready resource distinct lifetimes.
- Validate type, size, completeness and eligibility at the appropriate service boundaries. Clean incomplete objects and expired attempts according to a bounded policy.
- Decide access for previews, downloads, exports and any signed link. A link's lifetime and the revocation contract must agree.
- Preserve success when only part of a batch fails. Retry only the relevant attempt or processing phase, and keep it correlated to the original resource.
- For a file already stored, let a drop or Move to action request a folder change by ID. Verify access to both source and destination and apply the resulting audience policy to the file and its previews.

## Borrow for this workflow

Use [Media](../media/app-guide.md) when files need rendition or playback processing. Apply the parent audience from [Forum](../forum/app-guide.md), [Messenger](../messenger/app-guide.md) or [Customer support](../customer-support/app-guide.md).

Google Drive Help “Learn more about access to Google files” describes Viewer, Commenter and Editor roles.

## Acceptance cases

- A transfer reaches its total and validation fails. The product shows the correct outcome and retains what the user can retry.
- An interrupted batch is submitted again. Existing successful files remain available and no duplicate attachment becomes visible.
- Move a stored file into a restricted folder through either input path. The accepted result retains file identity and applies the intended access and preview rules.
- Revoke access and check the original, preview and any previously issued download reference against the agreed expiry or revocation behavior.
