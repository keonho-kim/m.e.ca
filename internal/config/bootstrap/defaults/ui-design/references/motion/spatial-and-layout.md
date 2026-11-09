# Spatial continuity and layout changes

Use spatial motion when it explains that the same object moved, expanded or gained context. Preserve actual identity, layout ownership and the reading/action path. A similar shape is not enough to claim that two unrelated records are the same object.

| Recipe | Useful relationship | Mechanics and guardrail |
| --- | --- | --- |
| Shared Element | A real collection item opens its detail | Match actual object ID and suitable visual element; do not duplicate active content |
| Container Transform | A create button/card becomes a related task surface | Interpolate a visual container, then expose the real named editor with deliberate focus |
| Inspector Reveal | Selected object opens beside a working collection | Reserve/reflow a useful track; retain minimum work width and correct selection |
| Layout Reflow | Same identified items change arrangement | Animate positions sparingly after real layout, preserving order, focus and selected identity |
| Reorder | User or actual service changes sequence | Explain the real order change; provide keyboard/non-drag path and conflict recovery |
| Intrinsic Expansion | Disclosure/help/result gains content | Use real size, stable text and natural final layout rather than distorted scale or guessed height |
| Split Resize | A desktop editor/preview division changes | Direct usable size changes and useful minima; do not trail behind pointer movement with long easing |

## Design the smallest meaningful moving region

Move the object or layer whose relationship changed, not every surrounding panel. A photo can retain identity from collection to detail while its metadata adopts a reading layout; a selected slide can replace its canvas while the tool frame and filmstrip remain stable. Missing or unrelated targets should use a straightforward alternative rather than a manufactured shared shape.

Use scale and origin to clarify hierarchy without distorting essential text. Direct manipulation needs immediate spatial feedback; later confirmation or persistence can be shown separately. Accent/selection and focus continue to identify the real object while motion is disabled or interrupted.

## Geometry and identity

Measure source/destination in a consistent coordinate space, accounting for scrolling, zoom and container transforms. A position-animation approach such as FLIP can compare real before/after geometry and animate an inverse visual transform, but it is not automatically needed. Reuse the project's layout primitive and avoid a new measurement engine for one panel.

Batch necessary measurements around real layout changes rather than alternating reads/writes each frame. Text should be laid out at its useful final size; aggressive container scale can blur or stretch labels. Keep clipping/shadow/rounded corners coherent without forcing expensive paint everywhere. Missing/unmounted targets, incompatible shapes or a changed object should take a direct/fade fallback.

Shared elements/snapshots represent one object visually; they do not own draft state, permissions, routing or form submission. Never clone a live form or expose an old protected preview after a workspace change. A source removed mid-transition needs a valid destination/focus fallback, not a retained hidden interactive source forever.

## Reordering and direct manipulation

Use stable IDs rather than row indexes. During drag/resize, keep controls and pointer capture coherent; an expressive spring cannot make values lag far behind direct input. Separate transient position from actual committed order and follow the server's conflict/partial-failure semantics. Animation cannot determine successful persistence.

For a layout/view switch, keep relevant query, selection and return context. Do not animate hundreds of offscreen items, replay entrances on virtualized mounts or keep both whole collections in memory solely for a morph. Measure large-data rendering before adding more choreography.

Reduced motion keeps final layout and meaningful selection with direct changes. Clean up observers, captures, frame callbacks, snapshots and retained source references. Test scrolling during transition, resize, changed data, removed source, rapid reversal, keyboard reorder and error recovery. Inspect actual desktop geometry and input responsiveness; a smoothly moving demo card does not verify a data workspace.

Sources: [MDN View Transition API](https://developer.mozilla.org/en-US/docs/Web/API/View_Transition_API) and [web.dev animation performance](https://web.dev/articles/animations-guide). FLIP and other names identify optional techniques, not product dependencies.
