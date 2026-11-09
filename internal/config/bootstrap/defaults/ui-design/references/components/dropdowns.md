# Dropdowns: actions and value choices

“Dropdown” describes presentation. Choose semantics before animating: an action menu runs commands; a select/listbox commits a value; a combobox adds suggestion/search behavior; a multi-select holds several values; an arbitrary filter form needs a suitable popover/dialog rather than menu semantics.

Anatomy: persistent label/trigger, current value or action context, resolved popup placement, option/action grouping, relevant selection/focus, applicable pending/empty/error and dismissal. Labels are not stable object IDs. Highlighted candidates are not necessarily committed values.

Consider whether choices should be visible for comparison, searched from a large set or inspected in richer context. A few consequential alternatives may fit radios or an explicit selection region better than a compact dropdown. The list below does not determine that decision.


## Choose visibility before compactness

Visible radios or a comparison region can serve a few consequential alternatives better than a dropdown. A large set may need search, meaningful grouping or a richer selection surface. Long content needs more room, not tiny type.

Keep the question/current value on the trigger and distinguish options, groups and secondary descriptions with type and spacing. Use consistent selection emphasis; category colors can support real categories, but options should not become an arbitrary rainbow. Focused candidate, committed value and disabled option must remain distinguishable.

A compact action menu and a value selector can share geometry without sharing behavior. Resolve their semantic job before choosing fade, scale or stagger. Frequent commands should not disappear into overflow merely because the menu looks polished.

## Adaptable treatments and optional motion

| Possible treatment | Use | Effect and constraint |
| --- | --- | --- |
| Native / Quiet Fade | A manageable known value set or ordinary menu | Keep native behavior where useful; custom appearance does not justify rebuilding keyboard logic |
| Anchor Scale | A compact menu/choice tied to the trigger | Subtle scale/opacity from resolved origin, with stable readable text |
| Directional Panel | Space places the popup above or beside the trigger | Enter from the actual side and constrain height; do not animate through a clipped table |
| Staggered Dropdown | A short contextual action group | Resolve placement and keyboard semantics immediately; cap a few item offsets, make focused items immediately visible and retain a static reduced-motion state |
| Submenu Slide | Genuine nested action groups | Short lateral reveal with stable parent path and keyboard access; no endless cascades |
| Searchable Panel | Large or remote option set | Keep query and results distinct; fade/reflow results only after current response identity is accepted |
| Token Multi-select | Several selected resources | Chips insert/remove with bounded layout motion; selection/removal names and IDs remain correct |
| Rich Selection | Objects require meaningful comparison | Use a suitable list/dialog with clear fields rather than an overloaded tiny menu |

Do not stagger every option before it becomes usable, turn typeahead into a decorative typewriter, or shrink labels to make a popup fit. Selected, highlighted/focused, disabled and pending remain distinct over the actual material.

## Keyboard, data and dismissal

Use native controls or the existing accessible menu/select/combobox. Preserve its arrow, Enter/Space, Escape, Tab and typeahead model rather than mixing models. A menu item with check state is a menu action pattern; it is not a substitute for any arbitrary multi-field form. Provide keyboard access to submenu return and selected-token removal.

Separate typed query, displayed label, highlighted suggestion and committed ID. Resolve an existing selection even if absent from the first page. Handle removed/denied choices and parent-value changes. Respect Korean IME composition; Enter ending composition cannot commit the wrong option or submit the enclosing form.

Cancel or reject obsolete remote results; debounce alone is insufficient. No matches differs from failed lookup or unknown availability. A new query must not label retained old options as current. Keep dependent loading local, with a meaningful retry where supported.

Exit animation cannot leave a dead overlay intercepting input. Closing/reopening reverses current visual work without obsolete focus restoration; an anchor removed mid-exit needs a logical fallback. Reduced motion uses immediate placement/selection without sliding a menu across the screen.

Test near-edge placement, long Korean options, remote response order, keyboard escape, removal of selected items, nested layers, rapid open-close and focus return. Own cleanup of positioning observers, input timers and requests. Do not preload every option or virtualize a short menu without a measured reason.

Sources: [WAI APG menu button](https://www.w3.org/WAI/ARIA/apg/patterns/menu-button/), [combobox](https://www.w3.org/WAI/ARIA/apg/patterns/combobox/) and [Radix Dropdown Menu](https://www.radix-ui.com/primitives/docs/components/dropdown-menu).
