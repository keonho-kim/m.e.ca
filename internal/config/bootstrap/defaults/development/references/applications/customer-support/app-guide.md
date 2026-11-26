# Customer support and ticket queues

Use this when a request needs intake, assignment, communication and a recorded outcome. Trace a request from the originating channel to the customer-facing and operator-facing views.

[Do & Don’t](do-and-dont.md) · [Implementation Q&A](implementation-qa.md)

## Main flow

1. Receive a request, confirm the requester and create or identify the ticket. Show receipt according to the channel and known outcome.
2. Assign an eligible operator or group and show the current status. Keep ownership and queue filters understandable during reassignment.
3. Select the response audience before sending. A public reply reaches the requester according to the channel. An internal note remains available to eligible staff.
4. Resolve the ticket with a meaningful outcome. Handle follow-up and reopen according to the product's rule for the ticket's state and originating channel.

## Implementation boundaries

- Keep ticket identity, requester, assignee or queue, status and message visibility explicit. Use the current access policy when accepting a send or assignment.
- Version assignment and status updates or apply a documented conflict rule. Track duplicate inbound events with stable channel identities where a retrying channel exists.
- Keep message persistence, external delivery and ticket resolution observable as separate outcomes when the channel can fail.
- Scope attachments, searches, exports, notifications and threaded history to the message's audience. If the workflow has deadlines, define the clock and which transitions affect it.

## Borrow for this workflow

Use [Messenger](../messenger/app-guide.md) for reply ordering and retry handling. Use [File sharing](../file-sharing/app-guide.md) for protected attachments and [Project work](../project-management/app-guide.md) for an issue handed to another team.

Zendesk Help “Adding comments to tickets” describes choosing between Public reply and Internal note.

## Acceptance cases

- A staff member adds an internal note with an attachment. The requester cannot read it through exports, email previews or download routes.
- The same inbound event arrives twice during reassignment. The ticket records one event and a permitted assignee.
- A customer replies to a resolved ticket. The selected reopen or follow-up rule is applied and visible.
