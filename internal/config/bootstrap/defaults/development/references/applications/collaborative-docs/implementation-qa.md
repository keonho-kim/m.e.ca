# Collaborative documents implementation Q&A

## Does autosave require a collaboration engine?

A queued save against a base revision can serve independent editors when a detectable conflict is acceptable. Choose merging or concurrent operations when the task needs multiple active editors to preserve compatible changes. State how the product handles an unmergeable edit and verify simultaneous saves.

## What should restore do to history?

If history is a product feature, restore content as a new current revision and record which source revision it came from. Decide separately how comment anchors, permissions and any published selection are affected. Show the outcome after the actual write is accepted.

## How should an editor handle a removed comment anchor?

Keep the comment linked to its document and available to the permitted audience. If the product supports anchored history, show its original context or mark it as detached and let an editor reattach it. Avoid retargeting by positional coincidence.

## What does saved mean with concurrent edits?

It means the relevant edit was durably accepted under the editing contract. Display the accepted revision or equivalent acknowledgement. A transport reconnect can fetch the current state and reconcile local pending edits without turning another user's successful save into an acknowledgement of this one.
