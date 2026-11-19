# Messenger implementation Q&A

## What should happen after a send times out?

Keep the original text and its send identity. Ask for the outcome or retry through the same idempotent operation when supported. Once a stored message is found, replace the provisional entry with that record. An unknown outcome stays visible until it is resolved or the user leaves that workflow.

## What orders messages across devices?

Use a service-assigned position scoped to the conversation, or another stable order with a tie breaker. A local timestamp can show when composition occurred. Confirmed history and subscription events use the same logical ordering and deduplication rules. A gap prompts a bounded history fetch when the transport can miss events.

## When should a read cursor advance?

Choose explicit marking or a reach point in visible content, depending on the experience. Share the cursor across devices when it represents the account. Reject a stale cursor update that would rewind it. Implement an explicit mark-unread action with its own semantics if the product supports reversing the indicator.

## How does an attachment affect completion?

Choose whether a message can post while its attachment processes. If allowed, expose the attachment's own pending or failed state. If the attachment is required, retain the message draft until upload and eligibility checks finish. In either case, resolve file access through the message's visibility when opening it.
