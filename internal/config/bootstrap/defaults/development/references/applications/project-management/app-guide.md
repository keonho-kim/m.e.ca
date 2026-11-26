# Project work, Kanban and task tracking

Use this when a team creates work, assigns responsibility, changes stage and checks what remains. Trace one record from intake through delivery or reopening.

[Do & Don’t](do-and-dont.md) · [Implementation Q&A](implementation-qa.md)

## Main flow

1. Open a board or list in the correct project and view. Expose actual filters, active stage and the work item's status.
2. Create or edit a task with stable identity, meaningful owner and applicable due date or dependency.
3. Reorder within a list or request a stage change when moving between lists. Save the stage and intended ordering together when the board contract requires both. Drag, the context menu and the visible Move action send the same request for equivalent moves.
4. On refresh or conflict, reconcile the confirmed stage and sort position. Preserve unsaved fields and show who can reopen or finish the work.

## Implementation boundaries

- Define which status transitions, assignments and cross-project moves the user may perform. Verify source and destination access and what moves with the record.
- Keep stage, completion, priority and display rank distinct. Only include dependency or approval rules that affect a real workflow.
- Store move intent with a task ID and the intended destination or neighbors. Resolve concurrent moves from current board state instead of relying on displayed array indexes.
- Keep a drag preview separate from a saved stage. Recheck the actual destination's eligibility when the request arrives. On rejection, reconcile the confirmed column and order and explain why the move failed.
- Keep completion evidence and history scoped. A linked discussion or attachment can have its own access rules and lifecycle.

## Borrow for this workflow

Use [Messenger](../messenger/app-guide.md) for threaded discussion and [Forum](../forum/app-guide.md) for community suggestions converted into tasks. Use [Administrative workspace](../admin-workspace/app-guide.md) for scoped bulk actions and permissions.

Atlassian Support “Move cards or lists” describes moving Trello cards within and between lists.

## Acceptance cases

- Two people move the same card. The board converges to a valid stage and ordering and surfaces any conflicts the person can resolve.
- Move a card through drag and through the visible Move action. Both follow the same transition and permissions. A forbidden move leaves the task in its confirmed stage.
- A moved item disappears under the current filter. Preserve the operation result and move focus to a useful remaining control.
- A user tries to assign or move work into a project they cannot edit. The current state remains intact.
