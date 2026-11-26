# Forms, surveys and intake

Use this when the outcome is a structured response or request. Identify who fills it, the version they saw and the operation that confirms submission.

[Do & Don’t](do-and-dont.md) · [Implementation Q&A](implementation-qa.md)

## Main flow

1. Load a form version and any resumable draft. Keep labels, required conditions and format information available during entry.
2. Apply branching and dependencies from the current answers. Explain what changed when a path hides an answer or makes prior input ineligible.
3. Validate at relevant field interactions and final submit. On failure, preserve valid input and lead the person to the applicable error or missing step.
4. Acknowledge the accepted response. Show its reference, edit policy or next step where the person needs it.

## Implementation boundaries

- Bind a response to the schema and branching rules under which it was entered. Decide how new versions affect open drafts and prior responses.
- Validate applicable-field and permission requirements at submission. Define whether a hidden answer may survive for resumption or should be excluded from accepted results.
- Choose identity for public, invited and signed-in submissions. Define repeatability, idempotency and who can see or export responses.
- Keep submission, optional notifications and follow-up processing identifiable. A failed follow-up can leave a valid accepted response.

## Borrow for this workflow

Use [Offline fieldwork](../offline-fieldwork/app-guide.md) for queued submissions, [File sharing](../file-sharing/app-guide.md) for attachments and [Reporting](../reporting/app-guide.md) for interpreting results across form versions.

Google Docs Editors Help “Show questions based on answers” describes conditional section routing in Google Forms.

## Acceptance cases

- Change an earlier answer so a later section becomes inapplicable. Confirm the accepted response contains only the answers defined by the form contract.
- Submit after a form change or response timeout. The user retains their work and can establish the accepted version and outcome.
- Show or export results with a mix of form versions. Labels and meanings remain traceable.
