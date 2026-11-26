# Secondary actions, chips and status

Use overflow/context menus for genuine secondary actions, grouped controls for related work and labeled status for actual state. Do not hide the primary task in an unexplained kebab menu. Action target and selection scope remain explicit before execution.

Prioritize the actual next decision and supported state vocabulary. Keep frequent actions discoverable, suppress unnecessary repeated acknowledgements and use status to explain a real condition. Empty badges, color variety and generic action menus cannot substitute for task structure.

## Choose the control or indicator

| Variant | Composition and behavior |
| --- | --- |
| Toolbar | Group related, named commands with their stable scope. Keep common actions available and move secondary commands into labeled overflow when needed. Use normal Tab order for a simple group. A composite toolbar needs one Tab stop and arrow movement through the group, with native editing keys available inside the controls that use them. |
| Filter chip | Show the condition's label and applied state. Use a toggle or the relevant selection control. Keep the applied query consistent with the visible chip. |
| Action chip | Use a short, named command such as “Add date filter”. Activate the described action and show any resulting state at its owner. |
| Input chip or removable token | Show the chosen value and a separate named control such as “Remove North region”. Move focus to a nearby token or back to the picker after removal. |
| Badge or tag | Show the category, count or status with useful text or an accessible equivalent. Define the count scope and make an unknown count distinguishable from zero. Use a separate action if people need to filter or open details. |
| Bulk action toolbar | Show the selected count and scope alongside actual available actions. Keep pending and partial results tied to those item identities. |

Keep focus movement in a toolbar separate from running a command. A filter chip is keyboard-operable and shows the same applied state after focus leaves. Removing a value uses its named action and moves focus to the next token, previous token or picker. A passive badge stays in reading order as part of the relevant item.

## Distinguish meanings that share a shape

A category tag, selected filter, saved state, quantity and process outcome may all be compact, but they do different work. Name the role before choosing color, fill, mark or motion. Keep category colors stable and reserve actual severity/success meanings separately.

Prioritize frequent local actions visibly; overflow is for genuine secondary commands. Use type, proximity and grouping so the target and consequence are understood before activation. A quiet interface need not conceal its useful operations.

A changed icon or local status can be enough acknowledgement when the effect is already apparent. Use a durable message when the person must recover, remember or investigate. Do not add another toast or animated count simply to make the interface feel active.

## Adaptable treatments and optional motion

| Possible treatment | Job | Effect and constraint |
| --- | --- | --- |
| Overflow / Context Menu | Secondary commands for the actual record | Anchored entrance with stable target, meaningful groups and visible keyboard/pointer alternative |
| Toolbar Emphasis | A related set of task actions | Static grouping or quiet local selected-state change; do not create ARIA toolbar semantics without its keyboard model |
| Filter Chip | An actual toggle or applied condition | Fill/outline transition tied to real selection, with a readable label and optional named remove action |
| Token Reflow | Selected resources have add/remove actions | Bounded position changes with stable IDs and correct focus after removal |
| Status Crossfade | An actual record changed state | Replace label/icon from confirmed/pending truth; no random live pulse or fictional event |
| Count Update | A real quantity changed | Stable-width value update when useful; no decorative count-up posing as measured data |
| Partial Bulk Result | Some selected items succeeded | Preserve successes and identify failed/unknown targets, with supported per-item recovery |

Static category, count, status, removable token and selectable chip can share a shape but keep different semantics. Do not make all pills buttons or turn every row value into a saturated badge. Use text and optionally icon alongside semantic color; unknown counts are not zero and abbreviated values must not falsify scope.

## Operation and focus contract

Menu content is action-oriented; arbitrary forms/selection need their own pattern. Use the existing accessible menu's trigger, arrow/escape/focus-return behavior. Context click cannot be the only action route. Submenus need a real hierarchy and bounded depth, not a way to hide ordinary frequent actions.

Verify current target, permissions and destructive consequence at execution. Client disabled styling is not authorization. Do not use an animation callback to decide whether a command runs, or let a later selection change retarget an already proposed operation.

Bulk scope is current page, explicit IDs or all matching records according to the actual contract. Partial failure is not full success. Retry/Undo uses the supported reconciliation/reversal path, preserves identities and does not overwrite unrelated newer changes.

Motion stays local to meaningful change. Reduced motion retains direct labels and selection. After token removal move focus to a logical neighboring control. Keep the outcome of pending work available. Test scope changes, late results, nested-menu Escape, removed targets, long status text and repeated live events. Bound notifications and release transient listeners and effects.

Sources: WAI APG menu button, toolbar and button.
