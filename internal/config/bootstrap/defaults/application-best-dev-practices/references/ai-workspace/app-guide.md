# AI conversation and task workspace

Use this when a person exchanges messages with an assistant, follows streamed output or starts work that may use tools or run in the background. Identify the durable outcomes the user needs to inspect or recover.

[Do & Don’t](do-and-dont.md) · [Implementation Q&A](implementation-qa.md)

## Main flow

1. Open a conversation and choose the relevant context, files and allowed actions. Give each person and workspace access to the intended records.
2. Accept the request once and start an identifiable run. Show available partial text or progress and distinguish proposed, approved and completed actions when those states exist.
3. Persist and deliver the accepted result, including available references and tool outcomes. Keep partial work inspectable according to the product's recovery policy.
4. On reconnect, cancel or retry, reconcile the durable run and actual side effects before starting another mutation.

## Implementation boundaries

- Separate conversation, request, run, generated answer and side-effect identities. A restarted presentation stream can recover the same run.
- Keep tool execution bound to current user authority and the approved target when approval is required. Treat retrieved documents and model output as input to validate at the actual execution boundary.
- Derive progress and cancellation from owned work. Distinguish a request to stop, its acknowledgement and already completed actions.
- Scope context, files, logs and result retrieval to the current requester. Give retained partial results and background work a defined lifetime.

## Borrow for this workflow

Use [Messenger](../messenger/app-guide.md) for history replay and deduplication, [File sharing](../file-sharing/app-guide.md) for protected artifacts, and [Project work](../project-management/app-guide.md) where an accepted result becomes a separately owned task.

MECA's usage guide describes streamed responses, tool progress and cancellation of work owned by a request.

## Acceptance cases

- Lose the stream after an action commits. Reconnect shows the durable result and avoids replaying its side effect.
- Cancel while text and a background action are active. Report the actual terminal and partial outcomes.
- Reopen an earlier conversation when a referenced file's access changed. Result and attachment retrieval enforce their current scope.
