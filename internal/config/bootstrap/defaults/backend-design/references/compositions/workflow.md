# Durable workflow across commit boundaries

Choose when accepted work must survive process/connection loss or an operation spans independently committed effects. A local transaction remains preferable when it covers the invariant. A [saga](https://learn.microsoft.com/en-us/azure/architecture/patterns/saga) coordinates local commits and business compensations; it does not provide a distributed rollback or isolation automatically.

## Concrete arrangement

```text
command → local transaction: business intent + job/outbox
dispatcher → bounded worker → external adapter
worker → conditional durable transition + next publication intent
relay → consumer: dedup identity + local effect in one transaction
status/subscription adapter → authorized projection of durable state
```

| Role | State and lifecycle |
| --- | --- |
| Workflow operation | Allowed transitions, guards, external command IDs and unresolved outcomes |
| Durable store | Job identity, current epoch/lease, checkpoints, conditional completion |
| Dispatcher/worker | Admission, execution, cancellation checks, result observation and cleanup |
| Relay/consumer | Publication acknowledgement, deduplication, per-object order and retention |
| Process root | Worker construction, readiness, bounded drain and shutdown joins |

Start with a state table and explicit step functions. Use a bounded Mediator/coordinator when several peers need one owner for progression. Use HSM for shared cancel/error transitions across real substates. An event-driven saga is appropriate only when ownership and causal dependencies stay understandable; do not hide the workflow in a growing web of observers.

## Intent, effects and compensation

Illustrative reservation states: `reserved → charge_pending → charged → confirmed`; failure may enter `release_pending` or `refund_pending`, then a resolved terminal state. Record an unknown external outcome separately from a known failed charge. Reconciliation must resolve it before refund/retry decisions.

Persist each step's identity before it can be retried. On redelivery, an idempotent consumer commits its scoped receipt and local mutation together. When the next effect is external, commit an intent and carry an identity that the destination can deduplicate/query. A transport acknowledgement is not proof of business completion.

Lease expiry permits reclaim according to policy, but does not stop an old worker. Use an epoch/fence checked by the authoritative write, and protect remote effects through their own identity/protocol. Cancellation is a requested transition; it cannot erase an already completed charge. Compensation may fail or require human resolution and needs its own durable identity.

A checkpoint resembles Memento only for the state it actually captures. Define version, replay boundary and retention; restoring it cannot reverse previously emitted emails/payments. A typed per-run observation record may act as a Blackboard, but observations need writers, freshness and scope rather than global mutable state.

## Cost and verification

Bound pending bytes, attempts, unresolved age, dead letters, active jobs and retained outcomes. Include outbox/consumer receipts and catch-up traffic in the storage and throughput model. Batch claims only while preserving fairness and short transactions; never keep locks around remote calls.

Inject failure after intent commit, after remote success and before local outcome commit. Resume two competing epochs and ensure stale completion is rejected. Verify compensation retries and shutdown while work is accepted. Keep ordering per object unless a stronger requirement justifies a global bottleneck. See [jobs/webhooks](../api/jobs-and-webhooks.md) for delivery details.
