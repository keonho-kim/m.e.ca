# Compose motion from the interaction's purpose

Names describe adaptable examples, not required effects or additional skill commands. Choose the information relationship first. Give logical state one owner, then derive the visual target. Animation completion never commits a mutation or determines task success.

## Navigation and contextual actions

| Composition | Useful context | Contract and adaptation |
| --- | --- | --- |
| Hamburger Morph + Side Drawer | Narrow workspace or genuinely secondary navigation | One real button owns aria-expanded/aria-controls; morph bars into a close cue while a single drawer enters from its actual edge. Desktop primary navigation can remain visible. Choose modal versus persistent policy deliberately; Escape/backdrop/close and focus return follow that policy |
| Side Drawer + retained work | Inspector, filters or occasional detail while reading a document | Preserve selected object and dirty draft. A nonmodal inspector must not trap focus; a modal mobile drawer must prevent access behind it. Layout shift is acceptable only when the composition actually reallocates space |
| Staggered Dropdown | A short contextual action menu | Establish geometry and semantics immediately; sequence only a few items with capped delay. First keyboard choice is usable immediately. Arrow/Home/End/Escape, outside close, viewport collision and removed opener still work. A value selector keeps selection semantics instead of pretending to be an action menu |
| Hamburger + drawer + submenu | A real navigation hierarchy | Do not chain three slow entrances. The drawer exposes the next destinations at once; a small submenu transition may explain grouping without delaying access |

Keep the total menu reveal short and make the first keyboard choice usable immediately. Do not multiply delay by an unbounded item count or clone focusable options for a snapshot. Timing is tuned at the motion-system owner; placement, clipping and focus remain with the actual popup primitive.

## Loading, content order and imagery

**Stagger + Skeleton:** reserve the known final structure while the first useful data is unavailable. Pick Pulse for a restrained rhythm or Shimmer for a deliberately directional cue. Mark decorative pieces aria-hidden and the actual region busy. On real success remove the loader, render stable IDs and optionally stagger only the first visible few regions. On error/cancel/missing data clear busy and provide the actual terminal state. Fast or cached content is not delayed, and refresh retains useful content.

Different regions can become ready independently: reveal a real heading or first data group while another region remains pending. Do not hide an already interactive toolbar until the last image decodes. A bounded stagger presents order; it does not serialize fetches or mutate array identity. Keyboard focus suppresses a visual hiding phase.

**Blur-up Image:** use a low-resolution derivative of the same image, reserve width/height or aspect-ratio, then reveal the decoded full source. A random gradient or unrelated photo is not a preview. Cached-ready images appear directly. Keep alt text meaningful once, preview decorative, and failure at the same geometry with a retry/alternate path where meaningful. Treat source changes, lazy loading and departure as owned lifecycles.

Do not pulse the skeleton, shimmer it, float the image and stagger every line simultaneously. Choose a coherent attention budget. A representative composition is static controls + Pulse skeleton for the content + one short bounded reveal + independently decoded imagery.

## Stages, progress and disclosure

Actual multistage work uses Step Status: queued/current/completed/error/cancelled/unknown from actual events. A progress bar measures known work; an indeterminate cue reports activity without a false percent. An input wizard/segment bar/breadcrumb indicates navigation or location. These may coexist but cannot substitute for each other.

An accordion can disclose Price Breakdown, use an Accent Bar for a contextual note, reveal a Step Timeline or show a chart. Preserve native semantics, intrinsic content size, announced state, focus and all relevant error inputs. Frequent comparisons often deserve simultaneous visible content rather than disclosure. Expand-all stays scoped to a meaningful group and handles mixed open states.

Screen replacement is a valid default. Swipe can clarify sibling or forward/back movement with a non-gesture control. Depth Stack can show a shallow parent/detail relation. Blinds/clip reveal is suited to a short noncritical inset, not repeatedly covering a working form. Shared element motion retains actual object identity. These effects should be omitted when they add waiting or imply a relationship that does not exist.

## Combine and verify

Specify trigger → logical target → rendered motion → interruption → final/reduced-motion state. Test rapid open/close/reopen, keyboard entry, source/query changes, failed/cached images, loading cancellation, back navigation, scroll/resize and departure. Preserve meaningful content and focus when animations are disabled. Release requests, animation handles, listeners, timers, observers and obsolete media.

The motion lab provides explicit presentation previews and real local loading; it labels the difference. The practical apps use the same rules on actual local operations. Service examples inform purpose and hierarchy, not proof that a brand implements each named effect.

Sources: [Apple motion](https://developer.apple.com/design/human-interface-guidelines/motion), [Material motion](https://m3.material.io/styles/motion), [WAI disclosure](https://www.w3.org/WAI/ARIA/apg/patterns/disclosure/), [menu button](https://www.w3.org/WAI/ARIA/apg/patterns/menu-button/), [MDN reduced motion](https://developer.mozilla.org/en-US/docs/Web/CSS/@media/prefers-reduced-motion).

## Choose a cue from the actual change

| Change | Possible acknowledgement | Retain |
| --- | --- | --- |
| A filter changes visible results | Short local replacement or no movement | One collection, correct scope and useful focus |
| A citation reveals evidence | Brief emphasis on the actual record | Source identity and reading context |
| A different slide/artifact is selected | Directional change or direct replacement | Stable tools and one active object |
| Saved state changes | A small mark/color change | Actual stored/pending meaning and a readable label |
| A computed amount changes | Stable value update, optional small emphasis | Real amount/units; no effect on unrelated keystrokes |
| An object is manipulated | Direct movement; optional inspector acknowledgement | Coordinates, selection and non-drag access |

These are examples of reasoning, not an event list or required API. Keep state independent of motion, interrupt obsolete effects and release retained resources. Reuse the project’s lifecycle mechanism. Native cross-document continuity is one optional implementation where supported, with direct navigation and stable object identity as the fallback.
