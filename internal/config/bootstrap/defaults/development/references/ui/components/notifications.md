# Toast, banner and result feedback

Match scope and persistence. A toast/snackbar gives brief acknowledgement; field/task errors belong near their cause; a banner explains a page/workspace condition; a confirmed result can need a lasting destination. No matches, no data, denied, failed and uncertain write are different states.

Anatomy: real operation/condition, concise message, meaningful severity, actual action/recovery when supported, appropriate dismissal and a durable route when necessary. Use more than color. Do not make a disappearing toast the only carrier of important recovery or a required action.

## Choose the surface and its parts

| Pattern | Choose when | Composition and behavior |
| --- | --- | --- |
| Local status | Frequent updates such as autosave belong beside a control | Use a short status label near the field or action. Update in place and retain focus. |
| Toast | Brief supplemental acknowledgement can coexist with work | Use concise outcome text and a visible × close button on every toast. Add one optional action when it is a useful shortcut. |
| Undo toast | The system can reverse an already completed action | Name the outcome and target. Show Undo and keep the × button separate. Give a useful reversal window and a durable route for essential recovery. |
| Inline alert | A task, section or item needs explanation or recovery | Place a title or message beside the affected region. Include the relevant correction, retry or result link and keep it visible while useful. |
| Banner alert | A condition affects the whole page, workspace or service | Keep scope and impact visible in a stable banner position. Offer only relevant actions. Allow dismissal when people can safely continue and find the information again. |
| Blocking alert | A time-sensitive decision must interrupt work | Use a dialog with a named action and safe exit under its focus contract. |
| Result or history | A partial, important or uncertain outcome needs later inspection | Show actual target, confirmed and unresolved items, and supported next actions. Link records where people return to work. |

## Make each toast dismissible

Always include a visible × close button, including on timed toasts and toasts with actions. Use a real button with an accessible name such as “Dismiss notification” and a reachable hit area. Keep it visible on touch and keyboard focus. Activating it dismisses immediately. A short optional exit fade can follow the logical close.

Keep the caller's focus when the toast appears. Leave action-bearing toasts available long enough to use, and pause automatic dismissal while a toast control has focus or the pointer is over it. If the close button has focus, move focus to the next available notification control or back to a logical task control before removing the toast. Clicking elsewhere can continue the task while the toast remains available. Use inline feedback inside an active modal so feedback is reachable within its focus scope.

An inline alert stays near its related operation while correction or recovery is useful. A page or workspace banner stays in a stable position across the affected context. Leave focus where people are working for routine updates. Announce meaningful results politely, and reserve interrupting announcements or dialogs for genuinely time-sensitive situations.


## Give feedback an appropriate weight

Place feedback near its cause and match persistence to what the person must remember. A changed saved icon or quiet local label may acknowledge an obvious reversible effect without another toast. A failed or uncertain commitment needs lasting context and recovery, even if a brief notification also appears.

Use severity color for real severity, paired with readable copy and an appropriate mark. Informational, selected and completed are different meanings. Avoid making routine success as loud as a blocking problem or using a large illustrated empty state where one useful sentence/action suffices.

Reserve a consistent notification area and enough space for long copy/actions. A banner can change page geometry once; a stream of toasts must not repeatedly cover the main task. Show the actual object or operation when several outcomes coexist.

## Adaptable treatments and optional motion

| Possible treatment | Use | Effect and constraint |
| --- | --- | --- |
| Quiet Fade | Routine acknowledgement | Brief opacity change; keep keyboard focus in the task |
| Edge Entry | A consistent notification area | Restrained translation from the chosen edge, without covering the primary action |
| Stack Reflow | A bounded set of separate outcomes | Animate surviving positions, preserve stable notification IDs and keep controls stationary while used |
| Dedup Update | Repeated updates about the same operation | Update the existing message/count with a small crossfade instead of accumulating identical toasts |
| Progress Handoff | An observed long operation becomes confirmed/failed | Replace status content from actual events; success does not follow elapsed display time |
| Inline Expansion | A save/import failure needs explanation or details | Expand a persistent task message while preserving input and a useful recovery path |
| Persistent Banner | Workspace-wide warning or connection condition | Reserve/reflow its area once; avoid repeated entrances that push the whole application |

Stacking is bounded by the product's actual needs; overflow must not silently lose material failures. Updates must not restart an endless animation or repeat live announcements for every progress tick. A decorative lifetime bar is not a job progress bar; use one only with actual dismissal semantics and an accessible durable alternative where needed.

## Actions, timing and accessibility

Do not steal focus for a toast. Announce routine status politely and reserve urgent interruption for genuine urgency. A visual alert and ARIA live `alert` are separate choices. Critical actions need enough time and a persistent route; if a timed notification offers interaction, follow the primitive's pause behavior on focus/hover and do not remove a focused control unexpectedly.

Undo requires a supported reversal, target identity and availability window. Handle expiration/failure honestly. Retry of an uncertain mutation requires status reconciliation or the actual idempotency contract; a friendly button cannot guarantee safe replay. Partial success retains confirmed outcomes and explains failed items.

Empty/result regions need useful task-specific text and actions rather than a generic illustration. Retry is for a retryable cause; create/reset/return depends on the state. Do not replace an otherwise usable app because a secondary region failed.

Reduced motion keeps static messages and immediate stack changes. Release expiry timers, subscriptions and animation handles on dismissal and teardown. Test notification bursts, changed targets, focus during dismissal, failed Undo, duplicate events, long messages, zoom and connection recovery. Report persistence once the actual operation confirms it.

Sources: WAI APG alert and Radix Toast. Reuse the project's current accessible notification implementation; these recipes do not require a new library.
