# Choose motion for component and layout changes

Name the change the user needs to notice. Choose the smallest cue that explains it while preserving attention, focus and useful work. Use a static update when the new state is clear on its own. Choose stronger motion for a meaningful spatial relationship or an expressive brief.

## Establish a temporal hierarchy

Match intensity to frequency and travel. Frequent controls benefit from quick or immediate feedback. Contextual layers can show their origin. A substantial change can use a short transition to support orientation. Keep useful content and focus available from the moment the action takes effect.

## Choose a cue for the actual change

| Change | Useful cue | Preserve |
| --- | --- | --- |
| Hover, focus or press | Immediate outline, fill or local press feedback | Visible keyboard focus and a stable hit area |
| Checkbox, switch or tab selection | Immediate mark or short indicator movement | Actual selection and independent focus, including rapid changes |
| Open or dismiss an anchored menu or popover | Short fade or small translation from the resolved anchor | Placement, keyboard entry and return focus. The first item is usable when opened. |
| Expand or collapse inline content | Immediate reflow or a short size change | Reading position, heading and usable adjacent controls |
| Open or dismiss a dialog or drawer | Quiet fade for a dialog or translation from a meaningful edge | Modality, dirty input and focus. A reopened surface follows the latest action. |
| Change page, panel or selected object | Direct replacement, short crossfade or shared-element continuity when identity persists | History, active content, selection and return context |
| Drag, reorder or resize | Follow input directly, with optional settling on release | Position, target, reading or tab order and a non-drag path |
| Refresh results or show work progressing | Retain useful in-scope content and show observed pending or measured progress | Applied conditions, accepted result identity and the actual outcome |
| Deliver feedback or a result | Immediate local status or a short reveal | Persistent recovery where needed and announcements at meaningful changes |
| Explain an occasional public story | A bounded reveal, stagger or other purposeful effect | Immediately reachable content and the actual next action |

Specify target, trigger, duration or physical behavior, final state and reduced-motion alternative. Match timing to the component's own rhythm and distance. Use the [timing and lifecycle guide](motion/timing-and-lifecycle.md).

## Read one relevant motion family

This map is level 2; each selected recipe is level 3. Component-specific variants live in their component recipes, so do not read every motion file to animate one control.

| Relationship to examine | Focused reference |
| --- | --- |
| Easing/spring, target state, reversal, cancellation and cleanup | [Timing and lifecycle](motion/timing-and-lifecycle.md) |
| Simple change, crossfade, Swipe, Blinds, Depth Stack or reveal between destinations | [Page transitions](motion/page-transitions.md) |
| Shared element, container transform, panel resize, reorder or layout continuity | [Spatial and layout motion](motion/spatial-and-layout.md) |
| Hamburger Morph, Side Drawer, Staggered Dropdown, entry + skeleton, Blur-up and Step Status composition | [Composed patterns](motion/composition-patterns.md) |
| Purposeful text/line/clip reveal, stagger, count-up, scroll narrative or celebratory accents | [Expressive effects](motion/expressive-effects.md) |

Recipe names describe alternatives rather than standards or new APIs. Conventional replacement is a complete valid baseline. Choose expressive motion for a real relationship or explicit brief, while retaining direct operation and a useful static/reduced-motion form.

## Keep state authoritative

Set the logical target state when the action occurs. Animation presents the change. Its completion callback must not be the sole mechanism that saves data, enables a control or marks work complete.

Test rapid open-close-open, changed selection, reversal, Escape, back navigation and departure mid-transition. An older callback must not close a newly reopened panel or overwrite a later choice. Do not retain invisible focusable content or a spinner after work ended.

## Fit the surface and theme

Operating screens need useful feedback, not repeated entrances on every row. Minimalism does not require motion to prove polish. Glass does not require a moving blur field. Futurism can express precision without fabricated liveness.

Use spring, bounce, parallax or stagger when the actual brief earns it. Bound total delay; do not make people wait for hundreds of items. Keep automatically moving content controllable and readable. A decorative count-up cannot substitute for progress from real work.

## Access, performance and cleanup

Honor reduced motion with the same final information and static feedback. Keep keyboard and pointer input usable and hit areas stable.

Start with transform and opacity when they express the intended change. Measure layout, paint, compositing and interaction before selecting expensive effects or adding optimization. Apply will-change only for an observed need and release it appropriately.

Reuse project tools and supported browser features with functional fallbacks. Release animation handles, timers, frame callbacks and observers with their owner. Verify reduced motion, interruption, slow data, failed work and representative desktop performance. A final-frame image cannot establish timing, gesture or cancellation correctness.
