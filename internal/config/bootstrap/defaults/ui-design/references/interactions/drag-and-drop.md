# Drag and drop

Use drag and drop to let someone propose a move, copy, insertion or file addition. Name the source, the effect and the eligible destinations. Give the same operation a visible click or tap path and an operable keyboard path. A Move button, destination picker or file chooser can serve both.

## Choose the drag pattern

| Pattern | Choose when | What to show and do |
| --- | --- | --- |
| Reorder within a list | The order is meaningful and editable | Give the item a recognizable handle when it contains other controls. Show the insertion line between real neighbors. Keep the original order until a valid release requests a move. |
| Move between columns or groups | The destination changes a category, stage or ownership | Name the destination and show eligibility, including an empty group that can accept an item. Preview its insertion position and expose a visible Move to group action. |
| Move or copy an existing item | A file, record or selection goes to another container | Preview the destination and make the effect clear before release. A dialog or menu can choose the same destination and effect. |
| File drop from the operating system | Dropped files begin an attachment or import workflow | Mark the intended zone, identify supported files and reveal any batch limits. Keep a visible file chooser. Create per-file pending and outcome states from the actual upload pipeline. |
| Place or move a canvas object | The position itself carries meaning | Keep the pickup offset and show the proposed position or snap guide in canvas coordinates. Provide meaningful position controls, a click-to-place option or a Move action suited to the task. |

For a list with a derived sort, offer a manual-order view or an explicit destination action when appropriate. In a filtered list, specify where the item is placed relative to actual neighbors, including items outside the visible filter. A drop target can expose a reason when the user can change the condition that makes it unavailable.

## Run one gesture through to its outcome

1. **Pick up.** Keep links, text selection and inline editing usable by starting from a handle or an intentional canvas mode. Record stable item identities, the source and the original position. Use the primary press and wait for deliberate movement before it becomes a drag.
2. **Preview.** Show one insertion or destination cue with a name or position. Keep it accurate as the surface scrolls, zooms, virtualizes or changes. Resolve geometry and hit testing in the same coordinate space, including the pointer offset and transforms. Start bounded edge scrolling only while the gesture needs it.
3. **Release or cancel.** On release over an eligible destination, submit one move, copy or position request, if the proposed value changed. An invalid release, Escape or a pointer cancellation before release clears the preview and restores the useful starting state. Treat lost capture during an active gesture as cancellation and release owned listeners and scrolling.
4. **Reconcile.** Show a pending move if saving is asynchronous. On acceptance, render the actual order or placement. On rejection, restore or reload the confirmed state, keep recoverable input and explain the next action. Recheck the same source, destination and permission contract for a drag, keyboard command or menu action.

Keep focus on the moved item or its useful control when it remains available. If the item is filtered out or the destination leaves the current view, move focus to a logical neighbor or named region and explain the result. Announce a meaningful settled outcome, such as a new position in a named column. Keep reading, visual and focus order coherent after a list move.

## Input, motion and browser behavior

- For keyboard users, provide named movement controls or the established grab, move, drop and cancel interaction with visible instructions. Keep arrow handling inside the relevant mode so editing and scrolling keep their normal keys.
- An explicit click or tap destination covers operation without dragging. Give it the same target rules, confirmation and failure recovery. A keyboard shortcut can complement it.
- When using Pointer Events for a custom in-page drag, scope pointer capture and touch behavior to the actual handle or interaction region. Preserve page scrolling and other pointer targets. Clear capture and any preview on cancellation, target removal or view exit.
- Keep the preview responsive to input. A small settle animation can show accepted movement. With reduced motion, preserve the same placement, focus and status with direct updates. Preserve element identity and recover focus when the browser or framework uses a fallback DOM move.

## File drop

Use the platform's file chooser and file-drop events as supported. Mark the actual drop zone only for an applicable transfer. For a native file-drop target, cancel its `dragover` where needed to accept the drop, then handle the drop without navigating away. Inspect the files on the drop and use the same validation, pending entries and outcomes as for the picker. The picker remains usable on devices without OS drag and drop.

Keep the destination clear before the drop, such as “Attach files to Incident 24”. State whether accepting the files starts transfer or adds them to a staged form. Identify each file's actual transfer, processing and error state, and retain accepted items when part of a batch fails.

## Verify representative cases

- Drag within and across lists, including empty groups, scroll edges, a blocked destination and a late data change. Repeat through a visible Move action.
- Cancel with Escape and pointer interruption. Verify the order and focus after cancellation and after a rejected save.
- Drop a batch with an invalid file and use the chooser for the same files. Check per-file outcomes and retained successes.
- Change canvas zoom and resize the view while using direct placement and its alternatives. Keep the object reachable and the final coordinate meaningful.

Principles follow the W3C WCAG Dragging Movements criterion, MDN File drag and drop, and Modern Web Guidance for drag and drop. For related resizing and gestures, see [Direct manipulation](direct-manipulation.md).
