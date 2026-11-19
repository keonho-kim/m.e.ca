# Timing, interruption and resource ownership

Describe motion as target + trigger + actual logical state + visual change + timing/physical behavior + interruption + final/static state. The same contract serves menus, fields, steppers and route transitions without a universal animation preset.

## Choose the temporal behavior

| Behavior | Suitable job | Constraint |
| --- | --- | --- |
| Immediate | Frequent state, reduced motion, input or costly content | Still show actual selection, result and focus |
| Ease-out entrance | A surface reaches a stable destination | Distance and duration should not postpone reading/operation |
| Short exit | Remove an already-understood surface | Clean up even when no completion event fires |
| Ease-in-out reposition | A stable object moves between meaningful positions | Preserve identity and interruption; never animate hit targets away during use |
| Spring | Direct spatial manipulation or intentional responsive material | Tune settling/overshoot to avoid unreadable controls and long tails |
| Bounce | A brief explicitly expressive accent | No habitual bounce for validation, menus or operational navigation |
| Bounded stagger | A small meaningful group reveals together | Total delay stays short and usable content/actions do not wait for their place |
| Loop | A genuinely pending/continuing condition | Real stopping condition, pause/teardown and static alternative |

Examples near 100–180ms for feedback, 180–260ms for a panel and 200–350ms for navigation can be starting points, not standards. Distance, frame rate, reader comfort, content and established tokens decide actual values. A spring's physical settings cannot be copied blindly into an unrelated CSS easing curve.

## Coherent temporal character

Choose motion by the product/region's intent: direct precision for repeated operations, restrained responsive material for a fitting choice, spatial continuity for related layers, or a bounded editorial rhythm for an initial group. Keep actual state and reduced-motion behavior identical. The expressive-craft reference in the visual-system map owns these positive cases.

A reading introduction, repeated schedule selection and direct object movement have different attention costs even inside one product. Define a small coherent timing vocabulary from those needs. A study’s custom properties, event owner or animation API demonstrate one implementation; use the target project’s equivalent rather than copying a global motion controller.

## State, reversal and completion

Logical open/selected/route state changes from the event or actual data, independently of presentation. Animation completion cannot be the sole path to saving, authorizing, submitting, committing a stage or reporting server success. A skipped, failed or zero-duration animation must still leave a complete interface.

For rapid open-close-open or A→B→A, cancel/re-target the current visual work or use the existing stateful primitive. Older callbacks must not close the latest surface, restore focus to a stale opener or commit an obsolete selected item. Avoid unbounded queues of entrances that replay outdated states.

An inactive layer can remain visible while exiting. Use the component primitive to keep it out of focus and hit testing. Restore focus, inert state and scroll state through an owner that handles interruption and route exit. Finish any deferred visual cleanup on cancellation and when an effect is skipped or has zero duration.

Cancellation of animation differs from cancellation of the task or network operation. Native canceled animation promises can reject; handle the expected cancellation path so it does not produce an unhandled failure or roll back confirmed domain state. If reduced-motion preference changes mid-effect, settle to the current logical target rather than restarting the old effect.

## Implementation selection

Reuse the project's timing tokens and primitives. Use CSS transitions for simple state changes and keyframes for a bounded sequence. Use an existing animation library or the Web Animations API when an interaction needs explicit sequencing, cancellation or retargeting. Check support for intrinsic-size and layout effects according to the target browsers.

For native top-layer dialogs and popovers, `@starting-style` can supply entry values. A CSS exit that stays in the top layer requires discrete transitions of `display` and `overlay`. Set `transition-behavior: allow-discrete` after the transition shorthand. Check both behaviors in supported browsers. Where the enhancement is unavailable, immediate close with correct focus, dismissal and final state is a functional fallback.

Render complete resting states for every preference. Enable optional movement under `prefers-reduced-motion: no-preference`. If a subtler effect still helps, use a short fade or a non-spatial cue and keep selection, focus and status visible under `reduce`.

Start with transform/opacity when they express the change. Measure necessary size, clipping, filter, shadow or layout effects on representative hardware; compositor-friendly properties still have memory/paint cost. Avoid per-frame layout read/write loops, large blur layers and permanent `will-change` on every control.

Cleanup owns animation handles, timer/frame callbacks, observers, listeners, pointer capture and retained snapshots. Hidden/background views do not keep decorative loops. Test reversal, Escape, back, route unmount, slow data, reduced-motion change, dropped frames and failed work. A static image or generic duration token is not evidence of correct timing.

Sources include MDN's reference entries for `Animation.cancel()`, `@starting-style`, `overlay` and `prefers-reduced-motion`, and Google's animation performance guidance.
