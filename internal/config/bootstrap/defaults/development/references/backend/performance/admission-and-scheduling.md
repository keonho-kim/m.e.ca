# Admission and scheduling as one resource contract

Design rate, concurrency, bytes and waiting time separately, then compose them. A request count cannot describe an export, small lookup and encrypted document decode equally.

| Mechanism | Controls | Does not establish |
| --- | --- | --- |
| Token bucket / rate window | Arrival volume and allowed burst | Bound on simultaneous slow requests or retained bytes |
| Concurrency semaphore | Number of active units | Equal resource cost or fairness |
| Weighted byte/cost admission | Estimated working resource use | Actual RSS bound or accurate CPU demand by itself |
| Bounded queue | Pending count/bytes and scheduling | More execution capacity |
| Tenant/work-class scheduler | Share and isolation | Global capacity unless budgets are coordinated |

## Concrete arrangement

```text
parse bounded metadata → authenticate/scope → estimate work class/weight
  → rate eligibility → bounded fair queue → reserve active capacity
  → execute → release exactly owned capacity → record outcome/cost
```

The admission owner defines rejection/timeout and resource leases. The scheduler owns queued state; a worker owns an active lease until cleanup. Reserve before large allocation. Cancelled queue entries must release references; a canceled worker must not return capacity while owned native work still uses it.

Set maximum weight so one accepted job can actually run. Avoid a queue whose first oversized job can never obtain capacity. Decide between FIFO, separate class queues or weighted/deficit scheduling from ordering and fairness requirements. Aging can prevent starvation but needs bounds; priority is not permission to discard already accepted durable work.

Per-tenant budgets combine with a global budget. A local limit of N on each of M replicas permits up to M×N aggregate work unless another authority coordinates it. Distributed rate accounting adds latency/failure behavior; define fail-open/closed or locally reserved quotas for the specific operation, not one policy for all endpoints.

## Feedback control is an optional refinement

Use fixed bounded admission first when it meets the requirement. Adaptive concurrency can react to measured dependency latency, but sampling, delayed feedback, missing failures and synchronized controllers can destabilize capacity. Keep hard memory/connection ceilings, minimum/maximum bounds, recovery behavior and an observable control state.

[Envoy's adaptive-concurrency filter](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/adaptive_concurrency_filter.html) demonstrates latency-based control and jittered minimum-RTT sampling. The linked `latest` may describe development behavior; verify the deployed version. It is a mechanism example, not a dependency or universal parameter set.

## Cost and checks

Observe intended arrivals, admissions, queue age/bytes, active weights, actual cost, rejected work and useful completions. Include cold-cache misses, retry budgets, rebuilds and slow clients. Reject cheaply before the saturated resource; an expensive denial path can itself overload.

Test a large tenant alongside small ones, one oversized job, many canceled waiters, a slow dependency, under-estimated expansion and shutdown while leases remain. Prove no capacity leak, premature release, cyclic acquisition wait or unbounded fairness state. Use [pipeline composition](../compositions/pipeline.md) for staged processing and [resilience](../operations/resilience.md) for deadline/retry ownership.
