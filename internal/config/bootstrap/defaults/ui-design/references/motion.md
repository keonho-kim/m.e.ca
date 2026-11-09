# Motion between application regions and states

Use motion to explain navigation, context, selection and actual work. Derive the cue from what the person needs to understand, not an effect name. The families below are examples to adapt or omit; a new composition can use the same state and accessibility rules without matching a listed recipe.

For the selected task, ask what the effect clarifies and what it costs in waiting, attention or interruption. Repeated operational actions and an occasional explanatory moment may need different treatment. Preserve a requested expressive identity while keeping content truthful and controls direct. Motion that only disguises empty content, fake activity or a poor workflow needs a structural correction.

## Establish a temporal hierarchy

Choose the intensity from frequency, relationship and attention. Direct manipulation follows the input; repeated operational actions settle quickly; a new contextual layer explains origin; an occasional editorial introduction can earn more rhythm. Those differences can coexist without giving each route a different animation personality.

Keep useful content and focus available immediately. A small accent in color or shape can acknowledge a real state without moving the whole region. If changing values are already clear, another toast, pulse or transition may add noise. Variety means choosing different useful cues, not cycling every recipe across the application.

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

## Select the transition by region

| Change | Useful cue | Preserve |
| --- | --- | --- |
| Page destination | Short continuity when it helps orientation | History, destination identity and appropriate focus |
| List to detail | A related transition or restrained shared element | Actual object and retained collection context |
| Inspector/drawer | Spatial origin and understandable layer | Modal policy, focus, dirty input and close behavior |
| Workspace resize or view switch | A clear reorganization | Readable data, selection and useful minimum widths |
| Query/refresh | Stable useful content and truthful pending feedback | Applied conditions and stale-response protection |
| Work progressing | Measured or properly indeterminate feedback | Real lifecycle, cancellation and completion semantics |
| Public product reveal | One purposeful explanatory or expressive moment | Content visibility, useful opening and actual action |

Specify target, trigger, effect, timing/physical behavior, final state and reduced-motion alternative. Small feedback might start near 100–180ms, a panel near 180–260ms and navigation near 200–350ms. Tune to distance, device and task; these are examples, not standards.

## Keep state authoritative

Set the logical target state when the action occurs. Animation presents the change. Its completion callback must not be the sole mechanism that saves data, enables a control or marks work complete.

Test rapid open-close-open, changed selection, reversal, Escape, back navigation and departure mid-transition. An older callback must not close a newly reopened panel or overwrite a later choice. Do not retain invisible focusable content or a spinner after work ended.

## Fit the surface and theme

Operating screens need useful feedback, not repeated entrances on every row. Minimalism does not require motion to prove polish. Glass does not require a moving blur field. Futurism can express precision without fabricated liveness.

Use spring, bounce, parallax or stagger when the actual brief earns it. Bound total delay; do not make people wait for hundreds of items. Keep automatically moving content controllable and readable. A decorative count-up cannot substitute for progress from real work.

## Access, performance and cleanup

Honor reduced motion with the same final information and static feedback. Keyboard and pointer input remain usable; no critical action depends on hover or a moving hit area.

Start with transform and opacity when they express the intended change. Measure layout, paint, compositing and interaction before selecting expensive effects or adding optimization. Apply will-change only for an observed need and release it appropriately.

Reuse project tools and supported browser features with functional fallbacks. Release animation handles, timers, frame callbacks and observers with their owner. Verify reduced motion, interruption, slow data, failed work and representative desktop performance. A final-frame image cannot establish timing, gesture or cancellation correctness.
