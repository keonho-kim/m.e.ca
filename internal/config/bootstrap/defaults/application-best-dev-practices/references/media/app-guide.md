# Media upload, processing and playback

Use this when people publish, play or resume audio or video, or when a shared file needs renditions. Identify which variant is usable at each point in processing.

[Do & Don’t](do-and-dont.md) · [Implementation Q&A](implementation-qa.md)

## Main flow

1. Load media metadata and allowed actions. For upload, verify the authorized source and expose transfer status and required fields.
2. Process the accepted source and report rendition or preview readiness. Where playback begins before all variants exist, show the supported quality or processing state.
3. Play, seek or resume under the applicable access and media version. Preserve a useful position across buffering, reconnect or recoverable failures.
4. Publish, replace or remove the media according to its lifecycle and visibility. Update playable and derived outputs according to the actual outcome.

## Implementation boundaries

- Separate upload, validation, processing, renditions, publication visibility and playback availability. Abort or retry the actual phase when supported.
- Use stable media and source revision identities. Scope progress and saved positions to the version and viewer where needed.
- Match authorization for manifests, captions, thumbnails, previews and playable segments to the promised access. Document lifetimes for already issued references.
- Budget processing and delivery from actual media volume. Choose cleanup, unsupported-input handling and per-rendition retry appropriate to the workflow.

## Borrow for this workflow

Use [File sharing](../file-sharing/app-guide.md) for transfer and protected access, [Content publishing](../content-publishing/app-guide.md) for release visibility and [Learning](../learning/app-guide.md) when playback contributes to a separate completion rule.

YouTube Help “Upload YouTube videos” and “Low video quality after upload” describe upload visibility and the availability of higher qualities during processing.

## Acceptance cases

- An upload finishes while higher-quality processing continues. Show playable renditions and the actual pending work.
- Replace a source during processing. Late results cannot supply variants for the new revision.
- Revoke a viewer's access and check playback requests, previews and issued references against the agreed rule.
