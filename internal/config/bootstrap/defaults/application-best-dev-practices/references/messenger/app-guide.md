# Messenger and team chat

Use this when people exchange messages in a direct conversation, group or channel. Begin with the ability to send, find and reply to a message in a conversation the person can access.

[Do & Don’t](do-and-dont.md) · [Implementation Q&A](implementation-qa.md)

## Main flow

1. Open a conversation and load a bounded history around its latest or unread position. Keep earlier messages reachable without moving the person away from what they were reading.
2. Compose a message or a reply to a specific message. Show a pending entry during submission. Attach it to the stored message when the operation acknowledges its ID and ordering position.
3. Apply new, edited and removed messages by ID and revision. Bring the person to incoming messages according to their reading position and notification preferences.
4. Reconnect from the last accepted position. Reconcile pending sends with history and offer a retry when the send failed or its outcome needs checking.

## Implementation boundaries

- Separate message acceptance, delivery and read state when the product exposes all three. Preserve a stable send identity across a resend after a lost response.
- Decide who can read old history after joining and which views remain available after leaving. Apply current membership at send, history, search, attachment and notification delivery.
- Bind reply, edit or removal from a contextual menu to the invoked message ID. Use the same permissions and result semantics as the visible action. If eligibility changes before activation, report the actual result for that message.
- Keep the account or recipient's read position scoped to the correct conversation and thread. Give typing and presence explicit expiry when those signals are part of the task.
- Choose the message retention, edit and removal policy together with quote and reply behavior. Remove derived previews in scope when access or content changes.

## Borrow for this workflow

Use [File sharing](../file-sharing/app-guide.md) for upload and protected downloads. Use [Project management](../project-management/app-guide.md) when a message can create a work item with its own assignee and completion state.

Slack Help describes replies grouped around a message in “Use threads to organize discussions”. “Use lists in Slack” describes adding messages to a list and item-specific discussion threads.

## Acceptance cases

- A send succeeds and its response is lost. A retry reconciles with the same message.
- Reconnect returns overlapping history and edits. Each message appears once with current content.
- A former member opens a saved route or attachment reference. Access follows the current membership policy.
- Open actions for one message while new messages arrive. Activation affects the intended message and follows its current edit or removal policy.
