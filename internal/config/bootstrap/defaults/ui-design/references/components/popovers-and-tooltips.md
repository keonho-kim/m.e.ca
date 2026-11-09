# Popover and tooltip

A popover anchors secondary content or a small interaction to a real source. A tooltip gives short noninteractive supplementary help on focus/hover. Important labels, errors and required instructions remain available without a tooltip. Interactive help needs an operable popover/dialog pattern, not focusable controls inside tooltip semantics.

Decide whether information is essential, frequently consulted or genuinely supplementary. Important context may belong inline or in a persistent inspector; do not move it behind hover or a small icon merely to achieve a visually empty surface.


## Keep source and explanation legible

The anchor should explain why this content is here. Use proximity, a stable edge and clear type before an elaborate pointer or origin animation. A compact surface suits a short explanation or bounded edit; substantial content deserves inline space, an inspector or a route.

Supplementary help can be quieter than the task but still needs readable ink. Essential price, permission, error and action information should remain available directly. Do not hide it behind hover to make a layout look minimal. A tooltip cannot compensate for an unnamed icon button.

Match material to the actual background. One floating layer can earn separation; nested frosted panels or repeated heavy shadows obscure hierarchy. On resize/zoom, keep content and its source relationship understandable rather than cropping the popup.

## Adaptable treatments and optional motion

| Possible treatment | Use | Effect and constraint |
| --- | --- | --- |
| Quiet Fade | Compact help or ordinary popover | Opacity change without moving the target or obscuring content |
| Anchor Scale | Quick action/detail visibly belongs to its trigger | Small expansion from computed anchor origin, including collision-adjusted placement |
| Directional Reveal | A panel placed above/below/beside its source | Short translation from the actual resolved side, not a hardcoded downward entrance |
| Hover Bridge | Supplementary preview that needs pointer travel | Stable entry/dismissal region; no flicker as the pointer moves from trigger to content |
| Shared Anchor | A related record preview | Restrained origin continuity while preserving one real target and no duplicate live editor |
| Inline Conversion | A narrow window makes floating interaction awkward | Use a suitable inline region/drawer with coherent focus and draft preservation rather than clipping the popup |

Names describe presentation, not ARIA roles. A popover can contain a menu, a form or explanatory content; each keeps its own behavior. A decorative arrow is optional. A missing arrow does not remove the need to identify the source.

## Placement and keyboard contract

Use the existing positioning primitive to resolve viewport edges, scrolling anchors, available size, zoom and overlay layers. Animate from its resolved side/origin. Keep placement stable during an ordinary entrance; expensive measurement every frame or oscillating collision flips create a different problem. Menus and tooltips cannot be cut off by a scroller's clipping workaround.

Interactive content needs a named trigger, keyboard entry/exit and appropriate focus return. Outside click alone is insufficient. Tooltip content does not receive focus; provide focus/hover discovery, dismissal and enough pointer stability to read it. Essential content must not depend on browser `title` behavior. Do not turn every plain information icon into a modal interruption.

When the anchor disappears, the selected row changes or a request returns late, close or rebind deliberately. An old preview's action must not target a new selection. An exiting popup must not intercept clicks while invisible or leave focus inside hidden controls. Reduced motion retains immediate content and stable anchoring without scale/translation.

Test keyboard open/escape, pointer bridge, scrolled/offscreen anchor, removed opener, rapid trigger changes, long Korean help, loading failure and zoom. Clean up positioning observers, timers, listeners and animations with the surface. Keep useful copy readable over the actual glass/soft/dark/light material.

Sources: [WAI APG tooltip](https://www.w3.org/WAI/ARIA/apg/patterns/tooltip/) and [Radix origin/collision-aware popover](https://www.radix-ui.com/primitives/docs/components/popover#origin-aware-animations). The APG tooltip page is draft guidance; verify the actual primitive and behavior rather than declaring a standard conformance result from this note.
