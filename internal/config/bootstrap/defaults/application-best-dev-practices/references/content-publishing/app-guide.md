# Content editing, review and publishing

Use this when authors prepare content and a public or shared audience sees a selected release. Trace one draft from authoring through preview, publication and later correction.

[Do & Don’t](do-and-dont.md) · [Implementation Q&A](implementation-qa.md)

## Main flow

1. Edit the draft under current author rights. Save the revision and keep a recoverable state when validation or storage fails.
2. Preview the intended revision and audience. If review exists, bind the decision to the submitted revision and identify any change that needs another review.
3. Publish or schedule a release. Check the current authority and selected revision and make it visible when the actual publication requirements are met.
4. Restore, supersede or withdraw a release under the product's policy. Update linked surfaces and keep the relevant author history.

## Implementation boundaries

- Give draft, review, published revision and public route explicit identities. Keep draft autosave and public visibility controlled at their own boundaries.
- Validate content and needed assets before switching the active publication when a complete release is required. Reconcile projections or caches according to the supported publication contract.
- Protect draft previews and shared assets with the intended audience. Handle scheduled edits, cancellation, attribution and the product's visibility or retention rules.
- For material from a restricted source, preserve its identity and revision for eligible editors. Verify what may be released to the destination audience, including files and generated previews. Define whether a source's withdrawal triggers review or withdrawal of a published copy.
- Define update, slug change, redirects, unpublish and restore semantics for search, direct access and already shared references where relevant.

## Borrow for this workflow

Use [Collaborative documents](../collaborative-docs/app-guide.md) for drafts and revision conflicts, [Media](../media/app-guide.md) for readiness and [Social feed](../social-feed/app-guide.md) when an audience receives a published entry. Use [Forum](../forum/app-guide.md) for promoting a discussion or accepted answer into a draft, and [File sharing](../file-sharing/app-guide.md) for checking the audience of included attachments.

WordPress Documentation “Revisions” describes comparing and restoring revisions of content.

## Acceptance cases

- Publishing fails during preparation. Verify the earlier public version and the author's recoverable draft.
- Edit a draft after review or scheduling. Confirm the actual selected revision and review policy on release.
- Withdraw one release and verify the page, share preview, search index and generated assets within the supported scope.
- Prepare a public draft from a private source. Changing or withdrawing the source follows the stated policy at review, release and retrieval.
