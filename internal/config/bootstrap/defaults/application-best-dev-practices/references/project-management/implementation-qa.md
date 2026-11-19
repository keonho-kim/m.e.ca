# Project management implementation Q&A

## What is the right way to store list order?

Use the project's existing position model when it satisfies sorting and concurrency. Small lists can save a scoped reorder transaction. Larger boards may benefit from stable rank keys and occasional rebalancing if workload warrants it. In both cases, define how the server places an item when a neighboring item just moved.

## How do optimistic moves handle failure?

Keep the pending target and last confirmed revision. On rejection, reconcile with the current record and related list positions. Restore focus and any unrelated edits. An older failed operation can only affect the proposal that originated it.

## Should an update from a thread change the task stage?

Link the discussion to the item and keep access scoped. Apply a stage transition only through an authorized action or a specified workflow rule. A reply can provide evidence without becoming the owner of task completion.

## How should filters interact with bulk moves?

Describe whether selection means visible IDs, the current page or all matching records. Freeze or version the intended scope at execution and recheck each eligible item. Report what moved and what failed when permission or stage changes during the operation.
