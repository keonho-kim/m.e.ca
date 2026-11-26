# Project management Do & Don’t

| Decision | Do | Don’t |
| --- | --- | --- |
| Transition | Define allowed source and target states and who can change them. | Let an arbitrary status string bypass an approval or completion rule. |
| Reorder | Persist the meaningful position and expose an equivalent non-drag action. | Make visual coordinates or a draggable index the durable order. |
| Concurrent edits | Check the current revision or apply a domain-specific merge. Keep the losing draft recoverable. | Overwrite newer titles, assignees or stages on every save. |
| Assignment | Resolve eligible assignees in the project and handle removed membership. | Assume displayed names are unique or always assignable. |
| Tracking | Give a discussion, task, checklist item and completion evidence their own agreed roles. | Mark a card done because someone visited or answered its discussion. |
| Filters and bulk actions | Make the actual selection scope clear and report per-item outcomes where needed. | Apply changes beyond the visible or explicitly selected scope. |
