# Popover and tooltip

A popover anchors secondary content or a small interaction to a real source. A tooltip gives short noninteractive supplementary help on focus/hover. Important labels, errors and required instructions remain available without a tooltip. Interactive help needs an operable popover/dialog pattern, not focusable controls inside tooltip semantics.

## Choose the anchored content

| Variant | Choose when | Composition and behavior |
| --- | --- | --- |
| Tooltip | A known control can use a short supplementary explanation | Associate concise help with the trigger. Show it on hover and keyboard focus. Keep focus on the trigger, allow Escape and let the pointer travel to the help without closing it. |
| Click or tap popover | More context or a compact interaction needs intentional entry | Use a named trigger and clear heading or opening context. Offer a close path and keep keyboard entry and return predictable. |
| Hovercard or preview | People benefit from a quick record preview before following a destination | Keep the anchor and preview associated. Provide explicit keyboard and touch access. Allow pointer travel to interactive preview controls and close on Escape. |
| Filter form popover | A few conditions can be edited while surrounding context stays useful | Include field labels, current values and a clear Apply when staged. Define what Escape and clicking outside do to drafts. |
| Inline help | The information affects completing the current task | Place it next to the subject or field and associate it with the control. Keep it available without opening a layer. |

Dismiss hover or focus help on Escape and when interaction leaves both its trigger and applicable hover content. An interactive popover opens from deliberate activation and can close from its trigger, Escape or a visible close action as appropriate. Keep a purposeful Tab path through interactive content. Return focus to the trigger when closing from inside, and respect a new focus target chosen by an outside click. Apply dirty-edit handling before dismissing staged input.


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

Test opening and Escape by keyboard, pointer travel, scrolled or offscreen anchors, removed openers, rapid trigger changes, long help text, loading failure and zoom. Clean up positioning observers, timers, listeners and animations with the surface. Keep useful copy readable over the actual material.

Sources: WAI APG tooltip and Radix origin/collision-aware popover. The APG tooltip page is draft guidance; verify the actual primitive and behavior rather than declaring a standard conformance result from this note.
