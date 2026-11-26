# Jobs, schedules and webhooks

Use when work must be deferred, retried or delivered independently of an HTTP connection. A queue changes ownership and failure semantics; it does not remove work or create exactly-once effects.

The [durable-workflow composition](../compositions/workflow.md) places state transitions, outbox, worker epochs, compensation and terminal observation. A simple job may need only a direct step and durable claim; hierarchical states and saga coordination are conditional.

## Claims and execution

Persist the job identity, input/reference, next eligible time and outcome authority. Claim a bounded batch atomically, then release the transaction before expensive work. Store a lease/epoch if jobs can be reclaimed and make stale completion fail at the authority. A heartbeat extends an observed lease; it cannot prove that an isolated worker stopped.

Define acknowledgement after the required durable effect, not after starting a goroutine. A crash after the effect but before acknowledgement causes redelivery; make the application idempotent. Kafka's [delivery discussion](https://kafka.apache.org/41/design/design/) explains the scope of transactional guarantees: arbitrary external destinations must cooperate.

Bound queue count/bytes, input retention, active jobs, attempts, maximum age, result size and dead-letter storage. Isolate interactive work from exports, reindexing and expensive decoders. Track oldest eligible age and useful completion throughput; queue length alone hides variable job cost.

## Retry and schedule semantics

Classify permanent input/permission failures, transient dependency failures and uncertain external outcomes. Retry with backoff/jitter inside a shared attempt/time budget; stop or reconcile unresolved effects instead of blindly replaying. Dead-letter entries need an accountable inspection/replay procedure and retention, not an infinite holding area.

Recurring schedules need an identity for each intended occurrence, timezone/clock semantics and a missed-run policy. Decide skip, coalesce or bounded catch-up. Include overlap, long runs, leader failover and clock changes. Do not let recovery emit an unbounded backlog into live capacity.

## Incoming events

Verify authentication/signature over the actual raw bytes before accepting semantic content. Bind verification to the right endpoint/account, check replay freshness as the protocol requires, and retain a scoped event identity. A valid signature is not permission for every state transition.

Commit durable acceptance before replying success if loss would violate the contract, then process asynchronously within policy. Order may differ from delivery order; resolve version/sequence or query authoritative state. Stripe's [webhook guidance](https://docs.stripe.com/webhooks) is a concrete example of signatures, duplicates, ordering and delivery behavior; its specific retry timing is provider policy.

## Outgoing delivery

Use committed publication intent, stable event identity, signing-key version, bounded payload and destination policy. A subscriber can receive duplicates; document acknowledgement and retry lifetime. Preserve per-object order only where required; global order can create unnecessary head-of-line blocking. Prevent customer-controlled destinations from becoming SSRF paths using [integration boundaries](files-and-integrations.md).

Verify crashes at claim/effect/ack boundaries, stale worker completion, duplicate events, reordered updates, invalid signatures, retry exhaustion and reconnect/catch-up storms. Include broker retention, consumer lag, outbox cleanup and dead-letter bytes in storage capacity.
