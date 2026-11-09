# Modal dialogs and drawers

Choose by interruption and task length. A modal requires a bounded task/decision before returning; a nonmodal inspector supports work alongside the current selection. A side drawer or bottom sheet can be modal or nonmodal: shape does not decide focus behavior. Substantial editors may need a route instead.

Anatomy: meaningful heading and target scope, content, clear actions, appropriate close/cancel, feedback and a return path. Destructive confirmation states the real target/consequence; ordinary cancel is not a danger action. Keep final fields/errors/actions reachable in long content and zoom.

Ask whether interruption helps this particular decision. Work that needs ongoing comparison may fit an inline region or inspector; consequential commitment may warrant a modal review. Elevation, backdrop or a fashionable material is not a reason to block the task.


## Design the layer around its decision

Make the task title, target, content and commitment readable as one bounded surface. Stronger elevation belongs to a genuinely interrupting layer; a persistent inspector can instead use a quiet edge and distinct working track. Do not choose modal behavior from a rounded card, blur or side placement.

Scale and density follow the content: a short confirmation, a detailed comparison and a long editor need different space. Keep necessary facts together and show the consequence before the primary action. A route or inline editor can be clearer when work is sustained. In narrow windows, preserve identity and action reachability rather than shrinking the whole panel.

Use accent for actual commitment/status, not the entire backdrop. Preserve the app's type and control vocabulary; a new surface does not need a new visual identity. Choose opaque content when material would weaken form or reading contrast.

## Adaptable treatments and optional motion

| Possible treatment | Use | Effect and constraint |
| --- | --- | --- |
| Quiet Fade | Ordinary confirmation or small dialog | Fade surface/backdrop together; do not delay access to the real decision |
| Focused Scale | A bounded new task needing local emphasis | Small scale plus opacity around a stable origin; avoid full-screen zoom or distorted text |
| Edge Slide | Inspector, settings drawer or deliberately modal sheet | Translate from the actual edge; keep selected object and remaining work width clear |
| Origin Expansion | A record/action opens a related editor | Expand from the true source toward a stable task surface; retain target identity and focus contract |
| Staged Confirmation | Review followed by submitting/result in the same bounded task | Transition body/status without moving action targets unpredictably; result needs real confirmation |
| Persistent Inspector | Frequent list/detail comparison | Reflow the canvas or reveal a reserved track; do not crush the table or reopen from scratch on every row |

Choose a consistent backdrop/surface hierarchy. Blur is conditional on material and measured cost; it is not modal behavior. A new drawer can slide while its body loads, but must show the correct selected identity. Never keep the previous record's form active under the new heading during a transition.

## Focus, exit and state

A modal moves focus appropriately inside, contains keyboard focus and makes background work inert. A nonmodal inspector needs a predictable route into and out of its content without trapping focus. On close, return to the opener if it survives; after deletion, use a logical neighboring row or heading.

Escape, backdrop, close, cancel, navigation and confirmed submit need deliberate dirty/pending behavior. Do not let one exit silently discard input or replay uncertain work. Use nested layers sparingly; define which closes first, owns focus and restores background state. Lock scrolling only with its owning surface and restore it on every exit.

Set open/closed and selected identity independently of the animation. An exiting surface cannot keep active invisible fields or block the background forever if an animation is skipped. Rapid reopen cancels stale close callbacks; unmount/cleanup must still release inert state, locks and focus handles. Defer neither mutation confirmation nor error visibility to an animation callback.

Reduced motion opens/closes immediately or with a short fade while preserving all semantics. Test rapid reversal, pending close, lost response, removed opener, row switch, long Korean title, nested scroll, zoom and route exit. Glass needs reliable text contrast; minimal/futuristic styling still needs ordinary labels and focus.

Source: [WAI APG modal dialog](https://www.w3.org/WAI/ARIA/apg/patterns/dialog-modal/). Reuse the actual library's focus and layer primitive; do not build a second focus engine for a visual variation.
