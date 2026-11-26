# Transactions, reservations and publication

Use when an invariant spans concurrent requests or a retry can repeat an effect. Write the invariant and the commit point before choosing isolation, locks or a queue.

The [transactional-command composition](../compositions/transactional-command.md) places the pure decision, scoped persistence and native error path around one commit. Use a [durable workflow](../compositions/workflow.md) only when independently committed effects require it.

## Put the decision at the authoritative boundary

| Invariant | Candidate mechanism | Failure to examine |
| --- | --- | --- |
| One business identity | Scoped unique constraint | Concurrent insert and same command with changed input |
| No lost update | Versioned compare-and-set | Zero affected rows, stale approval and retry recomputation |
| Resource cannot go below zero | Conditional update/constraint | Duplicate command and competing reservations |
| Several rows satisfy one rule | Transaction plus locking or serializable isolation | Write skew, deadlock and transaction retry |
| Job has one current writer | Conditional claim and fencing epoch | Paused worker writes after lease expiry |

For example, an inventory decrement can combine tenant/item scope and `available >= requested` in the update predicate, checking the result. Its reservation/command uniqueness must commit in the same transaction if retries must not decrement twice. A preceding availability `SELECT` is not the invariant.

PostgreSQL Read Committed uses statement snapshots; Repeatable Read does not by itself eliminate every serialization anomaly. Serializable failures require retrying the entire safe transaction with fresh reads. Distinguish a known rollback from unknown commit. See [isolation](https://www.postgresql.org/docs/current/transaction-iso.html).

Acquire locks in a stable domain order and keep the protected region short. Select the narrow lock that protects the invariant; an in-process mutex does not cover another server. Advisory locks require all participants and correct transaction/session scope. See [locking](https://www.postgresql.org/docs/current/explicit-locking.html).

## Model reservations as state

Represent reserved → committed/released/expired with explicit preconditions, version and ownership. Expiry observation alone cannot authorize a stale actor to commit. Bound outstanding reservations, reconcile uncertain external effects and make repeated release/commit harmless under the chosen identity.

Avoid a growing byte/count reservation held while waiting to acquire more of the same exhausted budget: several holders can prevent each other from completing. Reserve a known upper bound, acquire in a consistent order, or fail growth and release the current reservation. Cancellation and shutdown return only the resources actually owned.

## Publish without inventing atomicity

When a database mutation must generate an event, commit the mutation and outbox record together. A relay/CDC process can publish again after a lost acknowledgement; consumers apply an event identity/version idempotently and checkpoint deliberately. Debezium's [outbox router](https://debezium.io/documentation/reference/stable/transformations/outbox-event-router.html) is one concrete adapter, not a requirement to deploy Kafka.

For local durable job tables, claim a bounded set and commit the claim before executing external work. PostgreSQL's [`SKIP LOCKED`](https://www.postgresql.org/docs/current/sql-select.html) can avoid worker contention but gives an inconsistent view suitable for queue-like work, not general business reads or fair scheduling by itself.

Keep outbox retention, relay lag, failed delivery and deduplication lifetime bounded. Test commit-before-response loss, relay crash after publish, conflicting updates, expired claims and cleanup racing with active work. An outbox makes state plus publication intent atomic; it does not make an external consumer's side effect atomic with that transaction.
