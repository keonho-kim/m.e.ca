# Social feed and community discovery

Use this when people follow accounts, publish to an audience and discover updates across a changing set of authors.

[Do & Don’t](do-and-dont.md) · [Implementation Q&A](implementation-qa.md)

## Main flow

1. Choose the relevant feed, such as Following, a group or a public topic. Load a bounded, ordered slice and keep its scope visible.
2. Publish a post with a clear audience and attachment state. Make the result visible to eligible readers after the operation accepts it.
3. Apply reactions and follows as explicit user actions. Define what mute, unfollow, block and privacy changes mean on each affected surface.
4. When readers refresh, page or resume, add eligible entries without losing the current reading position. Show recoverable loading, empty and restricted states.

## Implementation boundaries

- Let the canonical post and current audience policy govern direct view, feed candidates, search, embeds and previews. A reshare carries a reference and follows the original's visibility policy.
- Decide where a relationship affects ranking, notifications or access. Removal from one feed and revoked access have different consequences through derived views.
- Use stable post identities and cursor semantics. If ranking can reorder items during a scroll, choose a defined snapshot or a refresh model.
- Choose fan-out, cache and invalidation boundaries from measured reader and publisher load. Define how stale content is prevented from resurfacing after a privacy change.

## Borrow for this workflow

Use [Forum](../forum/app-guide.md) for threaded replies and moderation. Use [Media](../media/app-guide.md) for pending processing and playback. Use [File sharing](../file-sharing/app-guide.md) when media retrieval must follow audience restrictions.

Mastodon Documentation “Posting to your profile” describes choosing visibility for a post.

## Acceptance cases

- A member changes an old post's audience. Test direct access, feed cache, search and notification previews under the new policy.
- A reader follows, mutes or blocks someone. The expected surfaces update according to that relationship.
- Overlapping pages or late updates arrive during scrolling. Each visible post keeps one identity and usable reading position.
