# AI workspace implementation Q&A

## How does a disconnected viewer recover a running task?

Read the run by stable identity and recover persisted history or a snapshot, then resume from the accepted sequence where supported. Deduplicate overlapping chunks and reject late updates from other runs. Show connection loss separately from the task's actual outcome.

## Does Stop mean the work ended?

Record the request to stop and attempt cancellation at each owned boundary that supports it. Report which work acknowledged cancellation, what already completed and which effects remain uncertain. Keep the partial answer and eligible results associated with that run.

## What should Retry do after a tool may have acted?

Inspect the original outcome and operation identity. Reconcile uncertain side effects before offering recovery. Starting another answer can use the existing result as context while a new external action receives the applicable scope and authorization.

## When should an AI result become an ordinary record?

Define the point where the user or automation accepts it. Attach provenance and the chosen revision, and let the domain's existing validation and permissions decide whether it can become a task, published document or other durable result.
