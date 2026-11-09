# Deadlines, overload and resource ownership

Design the degraded path with the normal path. A service should preserve useful completion capacity under excess demand instead of retaining unlimited waiting work. Choose bounds from the workload model and validate them under failure load.

## Put a budget around the whole operation

Set connection/acquisition, attempt and total-operation budgets deliberately. Propagate the remaining deadline through dependencies; include DNS/TLS, queueing and cleanup as applicable. Native gRPC clients need explicit deadlines because there is no universal safe default; see [gRPC deadlines](https://grpc.io/docs/guides/deadlines/).

An idle byte timeout, meaningful-progress timeout and total deadline answer different questions. A keepalive is not useful progress. Canceled request work should stop and release resources unless the contract explicitly transfers it to a durable server-owned operation.

## Admit work at the scarce resource

Bound global and per-tenant concurrency, queued bytes/items, accepted input and expansion. Reserve memory before allocating a large decoded object. Separate interactive and bulk capacity; admission based only on connection count misses expensive multiplexed requests. Use fair scheduling or weighted admission where request costs differ substantially.

Reject before spending the resource that is already saturated. Queueing needs an expiry, cancellation and shutdown path. Autoscaling is slower than an overload burst and may replicate pressure onto the same database. Amazon's [load-shedding account](https://aws.amazon.com/builders-library/using-load-shedding-to-avoid-overload/) and Google's [overload discussion](https://sre.google/sre-book/handling-overload/) show concrete production approaches.

## Control amplification

Retry only transient safe operations with backoff, jitter and a shared attempt/time budget. Count retries already done by SDKs, proxies and brokers. Keep ownership at a layer that knows effect uncertainty; a generic error handler should not replay commands. gRPC's [retry guide](https://grpc.io/docs/guides/retry/) also describes transport commitment and configured retry limits.

Hedged duplicate reads may reduce tails when requests are safe, spare capacity exists and the losing work can be canceled. They can amplify overload and cost; use a bounded hedge budget and representative validation. A paper or vendor result does not justify duplicating every request.

Circuit breakers can prevent repeated doomed dependency work, but half-open probes, state synchronization and recovery need testing. A fallback that sends all traffic to the primary database or returns different security/consistency semantics can worsen an outage. See Amazon's [fallback analysis](https://aws.amazon.com/builders-library/avoiding-fallback-in-distributed-systems/).

## Preserve ownership during shutdown

Stop admission, mark readiness appropriately, drain bounded in-flight work, cancel the remainder and join owned workers. Persist durable recovery state before releasing a claim whose outcome is uncertain. Release bodies, rows, timers, subscriptions and temporary files; do not detach goroutines that continue effects after shutdown reports completion.

Distinguish process liveness from readiness/capacity. Restart loops and routing traffic away from saturated instances can overload the survivors; Google's [cascading-failure discussion](https://sre.google/sre-book/addressing-cascading-failures/) explains that feedback loop.

Test slow/down dependencies, cache loss, cold restart, retry storms, slow consumers and shutdown during a committed command. Observe useful throughput, p99, queue age, retained memory, open resources and time to regain steady state—not only the number of returned errors.
