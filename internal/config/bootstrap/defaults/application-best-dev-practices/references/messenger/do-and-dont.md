# Messenger Do & Don’t

| Decision | Do | Don’t |
| --- | --- | --- |
| Send acknowledgement | Keep the draft and the pending entry until the accepted record or a recoverable failure is known. | Show sent after navigating away or after an animation ends. |
| Retry | Reuse the same send identity for the same attempt and check the recorded outcome. | Insert another visible message after every timeout. |
| Ordering | Use a stable conversation order and a cursor tied to it. Preserve the reading anchor when older messages arrive. | Sort solely by device clocks or jump to the bottom on every update. |
| Read state | Define what advances the cursor and qualify it by account, thread and conversation as applicable. | Mark a hidden page or an unopened thread read just because a frame arrived. |
| Notifications | Resolve the recipient and channel using current membership and notification preferences. | Deliver restricted text in a notification preview after access is removed. |
| Reconnect and edits | Apply each acknowledged revision at most once and reconcile pending data with accepted history. | Allow a late history page to undo a newer edit or put a removed message back. |
