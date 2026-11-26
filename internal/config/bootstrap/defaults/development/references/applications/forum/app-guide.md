# Forum, bulletin board and Q&A

Use this when a durable topic or question anchors a conversation people can revisit. Choose the board, category or workspace where posting and reading occur.

[Do & Don’t](do-and-dont.md) · [Implementation Q&A](implementation-qa.md)

## Main flow

1. List accessible topics with a defined sort, category and pagination. Keep pinned items and normal results consistent across pages.
2. Create a draft or post a topic. Give posted, awaiting review and rejected content a status the author can understand.
3. Open a topic, follow its replies and link to an eligible reply. Check current thread state and audience when accepting a posting or edit.
4. When moderation, deletion or an accepted answer changes the topic, reconcile the topic page, listings, search, counts and notifications.

## Implementation boundaries

- Give topics and replies stable IDs. Decide allowed nesting and how a reply retains its context after an ancestor is removed.
- Define reader, author and moderator authority for publishing, editing, locking, pinning, approving and hiding. Scope searches and reply links consistently with the current category policy.
- Model the accepted answer as a relationship to an eligible reply. Clear or re-evaluate it when that reply is hidden or removed.
- Decide whether public timestamps, revision history, moderator evidence and content retention follow different rules. Expose each only to eligible readers.

## Borrow for this workflow

Use [File sharing](../file-sharing/app-guide.md) for attachments under the post's access policy. Use [Project management](../project-management/app-guide.md) for turning an idea into tracked work, [Content publishing](../content-publishing/app-guide.md) for promoting an answer into a draft, and [Social feed](../social-feed/app-guide.md) for follow, mute and audience changes.

GitHub Docs “About discussions” describes Q&A categories and marking replies as answers. “Managing discussions” describes creating an issue from a discussion.

## Acceptance cases

- The author removes a reply with children. The chosen policy preserves necessary navigation and hides removed content.
- A moderator hides an accepted answer. The answer marker, direct link and search result follow the new visibility.
- While paging, a topic is pinned or moved. The visible set follows a clear ordering and scope.
