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

Presence during exit is a rendering choice, not permission for invisible interaction. Keep inactive layers out of focus/hit testing using the actual primitive while retaining appropriate exit visuals. Restore focus/inert/scroll-lock state through an owner that handles interruption and route exit, not only a fixed timeout.

Cancellation of animation differs from cancellation of the task or network operation. Native canceled animation promises can reject; handle the expected cancellation path so it does not produce an unhandled failure or roll back confirmed domain state. If reduced-motion preference changes mid-effect, settle to the current logical target rather than restarting the old effect.

## Implementation selection

Prefer existing tokens/primitives and CSS transitions/keyframes for simple property changes. Use an existing animation library or Web Animations API when controllable sequencing/reversal genuinely needs it. Do not introduce a motion dependency for one opacity transition. Intrinsic-size/layout effects need feature detection or measured fallback, not a blanket assumption about browser support.

Start with transform/opacity when they express the change. Measure necessary size, clipping, filter, shadow or layout effects on representative hardware; compositor-friendly properties still have memory/paint cost. Avoid per-frame layout read/write loops, large blur layers and permanent `will-change` on every control.

Cleanup owns animation handles, timer/frame callbacks, observers, listeners, pointer capture and retained snapshots. Hidden/background views do not keep decorative loops. Test reversal, Escape, back, route unmount, slow data, reduced-motion change, dropped frames and failed work. A static image or generic duration token is not evidence of correct timing.

Sources: [MDN Animation.cancel](https://developer.mozilla.org/en-US/docs/Web/API/Animation/cancel), [reduced motion](https://developer.mozilla.org/en-US/docs/Web/CSS/@media/prefers-reduced-motion) and [web.dev animation performance](https://web.dev/articles/animations-guide).
