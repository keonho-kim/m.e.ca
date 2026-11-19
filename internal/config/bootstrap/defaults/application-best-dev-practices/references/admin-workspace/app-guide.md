# Administrative and multi-tenant workspaces

Use this when operators manage records, members, roles or approvals within a workspace or organization. Make the actor, target and authority obvious at each consequential operation.

[Do & Don’t](do-and-dont.md) · [Implementation Q&A](implementation-qa.md)

## Main flow

1. Open the intended workspace and a scoped list or detail. Show applied filters, the target and the actions the person can currently use.
2. Edit a record, send an invitation or propose an approval. Preserve the draft and the exact target. Explain what will change and who can complete it.
3. Execute the requested change under current workspace and record authority. Show accepted, denied, conflicted or partially applied outcomes.
4. Review the current result and, where applicable, its authorized history. For bulk work or exports, show the scope and recovery at the same boundary.

## Implementation boundaries

- Enforce workspace and object scope on reads, writes, subscriptions, files and exports. The account's session alone does not establish access to every record.
- Bind invitations to their intended organization, permitted role, valid recipient policy and lifetime. Recheck eligibility on redemption and handle repeat or revoked invitations.
- Specify important authority invariants, such as retaining an eligible owner when a workspace depends on one. Bind an approval to the revision and scope the reviewer saw.
- Freeze or define bulk selection semantics at execution. Record attribution and outcome when the workflow needs a durable administrative record.
- Identify whether a row action applies to the invoked record or an explicit selection. Use its original workspace and record identities for actions opened by right click, keyboard or buttons. Verify current authority at execution.

## Borrow for this workflow

Use [Forms](../forms/app-guide.md) for editable details, [Project work](../project-management/app-guide.md) for review stages and [Reporting](../reporting/app-guide.md) for scoped preview and export.

GitHub Docs “Roles in an organization” describes organization and repository access roles.

## Acceptance cases

- A user changes workspace or loses a role after opening an editor. The next read, save and export apply current authorization.
- Redeem a canceled or already accepted invitation. It never creates a new or broader membership.
- Bulk approval targets a mixed set after a scope change. Each record and outcome remains identifiable.
