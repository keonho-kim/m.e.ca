# Offline fieldwork and synchronization

Use this when a person must capture or edit records without a reliable connection. Decide what can be completed locally, what must reach a service, and how the person knows the difference.

[Do & Don’t](do-and-dont.md) · [Implementation Q&A](implementation-qa.md)

## Main flow

1. Prepare the device with the permitted form, records or map data needed offline. Track which version and workspace the device has.
2. Capture data and attachments with stable local identities. Acknowledge local persistence and show which records are still unsent.
3. Retry queued work on reconnection, binding it to the original actor, scope and attempt. Acknowledge accepted records individually and keep errors or eligible conflicts actionable.
4. Refresh the relevant views and remove local data according to the product's retention and resumption rules.

## Implementation boundaries

- Give local draft, finalized queue entry and accepted server record distinct outcomes. Preserve a stable identity through retries and mapping to the server's record.
- Re-evaluate authorization, version and required files at acceptance. Plan for an expired session, removed assignment or invalidated local schema.
- Keep queues and caches scoped by account and workspace. Define handling of unsent work when users switch accounts or a device cannot retain the data.
- Choose a conflict policy from the affected fields and their risks. Keep a route to inspect unresolved changes before retrying or discarding.

## Borrow for this workflow

Use [Forms](../forms/app-guide.md) for captured schema version and branch rules. Use [File sharing](../file-sharing/app-guide.md) for retryable media and [Reporting](../reporting/app-guide.md) for counting records after acceptance.

ODK Docs “Helping data collectors take the right next step after filling a form” describes draft, finalized and sent forms in ODK Collect.

## Acceptance cases

- Restart the device after local save and retry during interrupted sync. The record remains recoverable and is accepted once.
- Change accounts with unsent work. A later sync uses the intended scope and a current authorization check.
- Submit a record with one failed required attachment and verify the visible server result follows the chosen completion rule.
