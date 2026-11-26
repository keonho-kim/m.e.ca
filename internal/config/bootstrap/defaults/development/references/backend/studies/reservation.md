# Study: inventory reservation and external payment

Fictional requirement: one command reserves scarce inventory and eventually confirms payment. Inventory is local transactional state; payment is an independently committed provider effect. Clients and webhooks can retry. No timeout may silently become a second charge.

## Local authority and composition

```text
inventory(tenant, item, available, version)
reservations(tenant, command_key UNIQUE within operation, item, quantity, state, version)
payment_steps(reservation, step_identity UNIQUE, provider_ref, observed_outcome)
outbox(event_identity, object_version, intent, delivery_state)
```

Use a [transactional command](../compositions/transactional-command.md) to bind scoped identity, check current eligibility, conditionally decrement inventory and record reservation/publication intent. The same transaction protects the local invariant; Redis cannot replace it. A [durable workflow](../compositions/workflow.md) owns payment observation, confirmation and compensation.

## Compare the boundaries

| Choice | Consequence |
| --- | --- |
| Call payment while holding local transaction | Long locks/connections; still no atomic commit with the provider |
| Commit reservation/intent, then call provider with durable step identity | Short local transaction; explicit pending/unknown states and reconciliation |
| One local transaction only | Best when the product has no independently committed external effect |

Assume 200 commands/s and an 80ms connection-holding interval around a remote call: stable average occupancy would be 16 connections from that interval alone. A hypothesized 5ms local transaction would contribute one. This arithmetic illustrates residence cost; it does not establish achievable throughput, lock behavior or a provider latency promise.

## Failure sequence

1. Commit `reserved` and charge intent with stable step identity.
2. Worker claims a bounded lease/epoch, calls the provider outside the DB transaction.
3. Provider succeeds but the response is lost: persist/retain `charge_unknown`; query/retry using the same provider identity according to its actual contract.
4. Only after a definitive outcome, conditionally confirm or release/refund. Webhook observation and active worker observation must converge under version/identity checks.
5. A stale worker cannot complete under a replaced epoch. Remote idempotency protects remote duplication; a local fence alone cannot stop a remote charge.

Cancellation/expiry needs a business policy for a payment already accepted or still unknown. Refund is a separate fallible command, not database rollback. Bound unresolved reservations by age/count/bytes and provide an accountable resolution path without deleting evidence.

## Evidence required

Test simultaneous same/different commands, oversell attempts, response loss after both local and remote commit, duplicate/reordered webhooks, expired worker resumption and refund failure. Check one local inventory effect per accepted identity, no duplicate external step and a recoverable status.

Budget outbox/receipt/outcome retention, retry traffic, active provider calls and per-item contention. Keep operation diagnostics centralized and emitted once at the actual outcome owner. A new service/broker is justified only by delivery/scale/isolation requirements that the local job/outbox arrangement cannot meet.
