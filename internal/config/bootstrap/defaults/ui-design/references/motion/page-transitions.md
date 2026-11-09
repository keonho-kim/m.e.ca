# Page and panel transition recipes

Choose by the relationship between destinations. Keep route/history, selected identity, title, focus, scroll restoration and real pending state authoritative. Simple replacement is the baseline; an effect cannot compensate for slow data or unclear navigation.

## Decide what needs continuity

A destination change, selected artifact replacement and opening an inspector are different events. Preserve the title/object and return context first, then choose whether direction, a shared visual or a simple change adds useful understanding. A long reader and dense work grid can share navigation semantics without the same entrance.

Do not animate the whole shell for a small local selection. Keep unrelated work stable and avoid a moving backdrop competing with content. Minimal or editorial styling does not require an authored transition on every page.

## Alternatives

| Recipe | Good fit | Mechanics and reduced-motion form |
| --- | --- | --- |
| Direct Change | Frequent operational navigation or no useful relationship | Render the destination directly with correct focus/scroll; already the static form |
| Crossfade | Same-frame peer views | Fade snapshots/visual layers, with one active destination; reduce to immediate or minimal opacity change |
| Fade Through | Dense unrelated content would overlap illegibly | Remove old visual emphasis before revealing new, without delaying available controls; reduce to direct change |
| Shared Axis | Ordered peer panels or parent/child navigation | Restrained common x/y/z direction reflects relationship; reduce to direct/fade |
| Swipe | Sequential stages, related slides or next/previous records | Translate along actual direction with stable frame; optional drag never replaces buttons/keyboard; reduce to direct/fade |
| Blinds | A deliberate section/story change earns a segmented visual reveal | A small bounded set of visual strips reveals the destination; reduce to one static surface |
| Depth Stack | A real hierarchy or layered preview has a clear return | Modest scale/translation and surface emphasis show entering/leaving a level; reduce to direct/fade |
| Anchored Wipe | A contextual reveal has an actual origin | Clip a noninteractive visual layer while the destination settles; reduce to immediate reveal |

These names are compositions, not required APIs or universal design standards. Swipe direction follows actual forward/back and writing direction; reverse navigation should not always replay forward movement. Depth Stack is not automatically a modal stack. A hierarchy effect must not invent background editors, blur away essential context or duplicate live routes.

## Blinds and depth without broken interaction

For Blinds, use a visual snapshot/mask or small bounded decorative layers rather than cloning the interactive DOM once per strip. The strips are not separate readable copies, focus targets or actions. Keep edges, clipping and final sizing coherent at zoom and resize; do not leave gaps or a permanently concealed destination if animation fails. Avoid repeated large blur/mask surfaces without measurement.

For Depth Stack, determine actual active level and return destination. Only intended content is interactive; hidden/covered surfaces cannot keep duplicate fields or intercept clicks. A background visual preview is not proof that its old data remains authorized/current. Keep scale modest enough for readable context and restore final geometry without persistent text distortion.

Swipe is an optional visual effect on desktop as well as a possible gesture. Do not take over the browser's back gesture, vertical scrolling, text selection or pointer input across an entire app. Provide ordinary navigation and cancel a drag without accidentally activating a child link. Use threshold/velocity behavior only when actual gesture navigation is implemented, not merely for an entrance animation.

## Route lifecycle and browser features

Use the existing router/transition integration. Where supported, View Transition API snapshots may provide continuity without two active copies of a page. Verify the specific same-document/cross-document feature and project/browser support; feature detection with a complete direct-navigation fallback is required. Do not claim universal support from one successful browser.

Keep navigation/data ownership outside animation completion. Do not put a long network wait into a blocking visual transition merely to hide loading. Destination loading/error states remain usable, and a rejected/skipped transition must not roll back a valid route. Respect supported history direction and scroll restoration rather than replacing them with a custom animation history.

Shared names/identities must be unique in the active transition and correspond to the same real object; skip the effect for missing/incompatible targets. Handle rapid navigation, skipped transitions, resize, removed elements and preference changes. Clear stale overlays/snapshots/listeners when their owner exits. A fallback must never leave content at opacity zero.

## Selection checks

For an ordinary SaaS queue, direct change or quiet fade may be sufficient. A real staged workflow can use shared-axis/Swipe. A deliberately expressive public product story may earn Blinds; a meaningful detail hierarchy may earn Depth Stack. Do not apply every recipe across one application to prove variety.

Test back/forward, direct/deep link, slow/error destination, rapid A→B→C, focused outgoing control, changed account, resize/zoom and reduced motion. Inspect interaction and memory with actual content, not only a final frame.

Source: [MDN View Transition API usage](https://developer.mozilla.org/en-US/docs/Web/API/View_Transition_API/Using). The effect recipes are original adaptations; no router or animation library is required by this skill.
