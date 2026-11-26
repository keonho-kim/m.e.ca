# Direct manipulation and gestures

Use direct manipulation when motion, placement or position helps someone work with visible content. Give every essential effect an equivalent visible control for click or tap and keyboard operation. Choose inputs from the task, the device and the surrounding scroll or editing behavior.

## Choose the interaction

| Pattern | Choose when | Default behavior and accessible path |
| --- | --- | --- |
| Split-pane resizing | Two work regions need adjustable space | Use a recognizable, named divider. Drag or use axis-appropriate arrows to change the pane size. Provide clickable presets or size controls and meaningful minimums. |
| Move a floating panel | Its position obstructs useful content | Use a dedicated handle and keep its contents operable. Offer dock or reset positions and reachable directional movement where needed. Keep the panel in view. |
| Object resize | A visible object has a meaningful adjustable dimension | Give the handle, dimensions and applicable bounds a clear purpose. Offer exact size fields or named adjustments. Preview constrained sizes and apply the one shown at release. |
| Swipe to reveal | A touch user needs fast access to secondary row actions | Reveal named controls and give the same row a visible actions button. Use an explicit action to commit, so previewing actions can be reversed. |
| Area selection | A bounded canvas or grid supports selecting several items at once | Show the selection rectangle, eligible items and count. Offer individual selection or a selection mode that can be completed without a drag. Keep the results attached to stable IDs. |

Use [Drag and drop](drag-and-drop.md) when release proposes moving or adding a record. Use [Context menus and panels](contextual-actions.md) for optional long press, right click and target-scoped commands.

## Resize or move with a predictable result

Retain the initial size or position, constrain the proposal in the correct coordinates and keep feedback close to input. Expose an adjustment handle with an accessible name and usable focus. For the window splitter pattern, give the focusable separator the name of its primary pane and expose its orientation, current value, bounds and controlled pane. Apply arrow behavior on its axis. Give collapse and restore their own behavior when the layout supports them.

Verify size limits and reachable actions at narrow widths and zoom. Re-clamp a moved or resized panel after the viewport changes. Offer a stable default for a floating surface. If both panes cannot meet their usable minimum, change the layout or offer a clear pane switch rather than stranding an editor or control.

## Preserve native interaction around gestures

Let the page keep its ordinary scrolling, browser zoom and text selection. Intercept only the intended pointer action and region. Cancel or yield when an action turns into scrolling, another gesture or pointer interruption. Stop any captured movement, animation frames and auto-scroll as the gesture ends.

For swipe-to-reveal, keep the current open state consistent for swiping and the visible actions control. Exposed actions become reachable when revealed. Closing removes hidden buttons from interaction and returns focus if it was inside. A scroll-snap implementation can enhance a supported layout, with the same actions still reachable through the button. Prefer the target project's primitive and preserve ordinary scroll when optional features are unavailable.

## Keep feedback and cancellation useful

Keep the logical value, the transient movement and the final confirmed state identifiable. During editing or direct manipulation, use immediate feedback. Limit follow-up motion to a useful settle or placement cue. Reduced motion keeps the same controls, scope and final value. When the product supports cancellation, restore the start value or the current confirmed state, with focus and selection intact where possible.

Verify at normal and reduced motion, using drag, click or tap and keyboard. Exercise overlapping scroll, lost pointer capture, quick reactivation, resizing below supported widths and a changed target. Check that a gesture preview never triggers a save, removal or route change by itself.

Principles follow the WAI-ARIA Authoring Practices Guide Window Splitter pattern, the W3C WCAG Pointer Gestures and Dragging Movements criteria, and Modern Web Guidance for drag and swipe to reveal.
