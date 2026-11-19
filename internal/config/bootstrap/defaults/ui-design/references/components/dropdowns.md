# Dropdowns: actions and value choices

“Dropdown” describes presentation. Choose semantics before animating: an action menu runs commands; a select/listbox commits a value; a combobox adds suggestion/search behavior; a multi-select holds several values; an arbitrary filter form needs a suitable popover/dialog rather than menu semantics.

Anatomy: persistent label/trigger, current value or action context, resolved popup placement, option/action grouping, relevant selection/focus, applicable pending/empty/error and dismissal. Labels are not stable object IDs. Highlighted candidates are not necessarily committed values.

## Choose a command menu or value picker

| Variant | Choose when | Composition and behavior |
| --- | --- | --- |
| Menu button | A labeled group of commands belongs together | Use a trigger with open state and grouped command or destination items. Open from activation, move by arrow keys and close on Escape or intentional activation according to the item. Restore focus on dismissal. |
| Meatballs menu | Repeated rows or cards need secondary actions | Use a visible ellipsis button named for its target such as “More actions for Report”. Position it by that row and retain target identity through refresh. |
| [Context menu](../interactions/contextual-actions.md) | The invoked item or point determines available commands | Open by secondary click or a keyboard context-menu gesture such as Shift+F10 where available. Anchor keyboard invocation to the focused item and provide a visible action entry. |
| Checkable menu or submenu | Commands include a quick display setting or a real nested group | Show checked state for the setting. Open child menus using the primitive's keyboard and pointer paths. Escape in a child returns to its parent item. |
| Native select | A known exclusive value fits a compact form field | Show a persistent label and current value. Preserve the platform's open, selection and dismissal behavior. Save with the form when staged. |
| Listbox or visible radios | Options should stay on screen for scanning or comparison | Use radios for a small, explained exclusive set. A listbox uses its option navigation and selection model. Put independent links and buttons outside options. |
| Editable combobox | Users search fixed choices or can accept typed text | Keep input and active suggestion distinct. Arrows move the suggestion, Enter accepts it and Escape dismisses it. State whether arbitrary text is allowed and when the value commits. |
| Multi-select or token picker | Several choices can coexist | Show checked options and selected tokens or a readable summary. Let users toggle without reopening after every choice. Give removal its own named action. |
| Filter popover or rich picker | Several fields or enough evidence to compare are required | Use a named form or larger selection surface. Show relevant facts and an Apply or Save action when edits are staged. |

Use individual buttons for commands and links for destinations. A disclosed group of ordinary navigation links keeps the normal Tab sequence. A composite menu supplies its arrow-key behavior.

For a button menu, Enter or Space opens it and keyboard entry focuses an available item. Up and Down move through items without activating them. Escape returns focus to the opener. Tab leaves the menu and closes it. An outside pointer action closes it without taking focus away from the clicked destination. A selected link follows its navigation path.

Use explicit acceptance for a suggestion that sets an object ID. In an editable combobox, keep normal editing keys available in the text field. Tab continues through the form. Closing the suggestion list preserves the previously committed value until a new value is accepted. Show directly when there are no matches or the lookup fails.

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

Separate typed query, displayed label, highlighted suggestion and committed ID. Resolve an existing selection even if absent from the first page. Handle removed or denied choices and parent-value changes. Let Enter finish active text composition. Use a subsequent deliberate action to choose or submit.

Cancel or reject obsolete remote results; debounce alone is insufficient. No matches differs from failed lookup or unknown availability. A new query must not label retained old options as current. Keep dependent loading local, with a meaningful retry where supported.

Exit animation cannot leave a dead overlay intercepting input. Closing/reopening reverses current visual work without obsolete focus restoration; an anchor removed mid-exit needs a logical fallback. Reduced motion uses immediate placement/selection without sliding a menu across the screen.

Test near-edge placement, long option labels, remote response order, keyboard Escape, removal of selected items, nested layers, rapid open-close and focus return. Own cleanup of positioning observers, input timers and requests. Do not preload every option or virtualize a short menu without a measured reason.

Sources: WAI APG menu button, listbox, combobox and Radix Dropdown Menu.
