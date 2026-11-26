# Modal dialogs and drawers

Choose by interruption and task length. A modal requires a bounded task/decision before returning; a nonmodal inspector supports work alongside the current selection. A side drawer or bottom sheet can be modal or nonmodal: shape does not decide focus behavior. Substantial editors may need a route instead.

Anatomy: meaningful heading and target scope, content, clear actions, appropriate close/cancel, feedback and a return path. Destructive confirmation states the real target/consequence; ordinary cancel is not a danger action. Keep final fields/errors/actions reachable in long content and zoom.

## Choose a surface by interruption and scope

| Variant | Choose when | Composition and behavior |
| --- | --- | --- |
| Confirmation modal | A bounded consequence needs review before commitment | Name the action, target and effect. Pair a specific confirm label with a safe Cancel. Focus the safe choice for an irreversible action. |
| Form modal | A short task can complete without using the background | Use a title, labeled fields, nearby errors and reachable Submit and Cancel. Route Close and Escape through the draft policy. |
| Nonmodal inspector or drawer | Comparison, editing or selection should continue beside the source | Show the current item's name, a close control and an obvious keyboard route in and out. Preserve list context and handle a target change deliberately. |
| Modal drawer or bottom sheet | A bounded task needs edge placement and exclusive focus | Keep a named heading and visible close or cancel action. Swipe can supplement these paths. Restore focus or move to the chosen destination. |
| Dedicated editor | The task is long, resumable or needs its own history entry | Give it room, keep draft state and present a meaningful return path. |

## Scrim and dismissal

Use a scrim to identify the boundary of a modal. Tie it to the active layer, using its backdrop where available. Keep it decorative and outside the focus order. The modal primitive supplies background inertness and focus containment. Choose whether clicking the scrim requests dismissal and apply the same draft or pending policy as Close. Provide a visible Cancel or close control and keyboard dismissal appropriate to the task. Make scrim and surface transitions follow the same logical open or close state, including immediate changes with reduced motion.

On modal entry, move focus to the first useful control, a readable heading for substantial content, or the safe confirmation action. Tab stays within the modal. Escape requests the same safe dismissal as Close, including the unsaved-change check where applicable. Commit from the named action. On dismissal, return to the opener or a logical nearby control. Successful navigation can move focus to its new destination.

For a nonmodal drawer, keep the background operable and allow normal keyboard travel. Keep a visible Close control and return focus logically when closing from inside. For an edge drawer with swipe dismissal, keep an equivalent button path and honor a cancelled gesture.


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

Use the target system's accessible modal or native `<dialog>` opened with `showModal()` where supported. Give the dialog an accessible name and move focus inside. Focus a safe action for a short, irreversible confirmation. A modal contains focus and makes background work inert. A nonmodal inspector needs a predictable route to and from its content while the background stays operable. Return focus to the surviving opener on close, or to a logical neighboring row or heading if it was removed.

Escape, backdrop, close, cancel, navigation and confirmed submit need deliberate dirty/pending behavior. Do not let one exit silently discard input or replay uncertain work. Use nested layers sparingly; define which closes first, owns focus and restores background state. Lock scrolling only with its owning surface and restore it on every exit.

Set open/closed and selected identity independently of the animation. An exiting surface cannot keep active invisible fields or block the background forever if an animation is skipped. Rapid reopen cancels stale close callbacks; unmount/cleanup must still release inert state, locks and focus handles. Defer neither mutation confirmation nor error visibility to an animation callback.

Reduced motion opens/closes immediately or with a short fade while preserving all semantics. Test rapid reversal, pending close, lost response, removed opener, row switch, long titles, nested scroll, zoom and route exit. Glass needs reliable text contrast. Minimal and futuristic styling still need ordinary labels and focus.

Source: WAI APG modal dialog. Reuse the actual library's focus and layer primitive; do not build a second focus engine for a visual variation.
