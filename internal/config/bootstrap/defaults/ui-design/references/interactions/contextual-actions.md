# Context menus and panels

Use a contextual action surface when the next action depends on a record, selection or place. A right click can request that surface. Provide a named visible trigger for keyboard and touch users, and apply the same target and action rules when they use it.

## Choose the surface

| Surface | Choose when | Conventional behavior |
| --- | --- | --- |
| Context menu | The user needs a short list of commands such as Rename, Duplicate or Move | Show actions scoped to the invoked item or surface. Use arrow navigation, explicit activation and Escape dismissal. Give the visible actions button a target-specific name. |
| Anchored action popover | One or two fields, a compact chooser or richer context is needed | Name the target, keep controls in normal Tab order, and show an Apply or Close action where needed. Set a draft and dismissal policy for pending or staged edits. |
| Nonmodal inspector or drawer | Reviewing details or editing takes longer and the surrounding work remains useful | Give the panel a heading, visible Close and a predictable route from the source into the panel. Keep the background operable and preserve the selected record or draft deliberately. |
| Modal confirmation | A sensitive or expensive command needs a bounded decision before work continues | Open from the deliberate command. Name its consequence, provide a safe Cancel, follow the modal's focus contract and confirm only through the intended action. |

The surface follows the task. A command menu can open an edit popover or request a confirmation. A revealed group of ordinary buttons has normal Tab behavior. Use menu semantics only for the composite keyboard model. Use dialog semantics for a named dialog with the associated behavior.

## Choose and preserve the target

A pointer-triggered menu uses the item under the pointer or the point on the actual workspace. A keyboard request uses the focused item. Capture the item ID, relevant workspace, eligibility and intended selection at open. Show the target in a label, heading or selected appearance. If a command applies to a group, name the selected count or scope. If the invoked object is outside the selection, make the single-item scope clear or update selection deliberately.

Check that the captured target still exists and is eligible at activation. A refresh, filter change or removed item can make the action unavailable or close the surface. Preserve recoverable panel edits with their original target. The previous selection and cursor coordinates never establish access on their own.

## Provide pointer, keyboard and touch entry

- Open the contextual menu from a scoped context-menu request on supported surfaces. Handle the platform's Context Menu key or Shift+F10 where available, and let the visible actions button open it with Enter or Space. Avoid handling a shortcut twice when it already produces the context-menu event.
- For pointer entry, place the surface near the invocation point. For keyboard and visible button entry, anchor to the item or trigger. Resolve viewport collisions and set a useful maximum size with an operable scroll path. Preserve an anchored position through the existing positioning primitive when CSS anchor positioning is unavailable.
- Keep useful browser menus available on editable text, links and unhandled areas. Cancel the native context menu only when the actual target has an applicable app action surface.
- On touch, keep the same named actions control available without a gesture. Long press can supplement it on an intentional target. Cancel on scrolling, movement or competing gestures, and preserve native selection and copying where those belong to the task.

## Manage focus and closing

- **Menu.** On keyboard open, move focus to an available item. Up and Down navigate. When a pointer opening keeps focus on a trigger, let Down enter the menu. Activation runs the chosen command for the captured target. Escape closes the current menu or submenu and returns to its opener or parent item. Tab exits the menu and continues navigation.
- **Popover or inspector.** Focus a useful heading or control when entering an interactive panel, and provide a natural Tab path. Use an explicit Close or Cancel for a substantial surface. Escape closes the appropriate active layer under the same dismissal policy. Use light dismiss for a surface where doing so preserves the task and input.
- **Return.** When closing from inside, restore focus to the invoking item, an available actions trigger, or a logical neighbor if the source disappeared. Preserve the new focus when the person uses a different control outside. A completed navigation can move focus to the destination.
- **Animation.** Open at the resolved point or anchor and make the content usable immediately. Keep a usable final state with reduced motion. In a fast retarget or reopen, an older exit cannot close the newer surface or restore focus to its old target.

## Verify representative cases

Open on a selected and unselected row, over empty canvas, near an edge, and after scrolling or zooming. Exercise the visible action path and keyboard entry. Remove or change the source while open, dismiss outside into another control and reopen rapidly. Check the target, first and returned focus, pending edits and the actual result of each action.

Principles follow the WAI-ARIA Authoring Practices Guide menu and dialog patterns, MDN's contextmenu event reference and Modern Web Guidance for menus, dialogs and viewport-aware dropdowns. For shell and layer details, see [Dialogs and drawers](../components/dialogs-and-drawers.md) and [Popovers](../components/popovers-and-tooltips.md).
