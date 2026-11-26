# Workload, cost and capacity

Use when selecting an architecture or evaluating a costly path. For a new service, build an explicit model; for an existing one, replace assumptions with observed distributions. Retain uncertainty instead of making an estimate look measured.

## Model the complete path

For each important operation, account for: admission → parse/auth → pool wait → query/lock → materialization → encryption/serialization → publish → client consumption. Include work after the response, retry attempts and maintenance. Count the input bytes, scanned candidates, decoded objects, copies, database round trips, durable writes and downstream calls.

Separate low-latency transactional requests, long streams, bulk jobs and rebuilds. A shared queue or pool can let a large export dominate interactive work even when average utilization is low. Bound each class and its contribution to the shared bottleneck. [Amazon's load-shedding account](https://aws.amazon.com/builders-library/using-load-shedding-to-avoid-overload/) describes protecting useful serving capacity when demand exceeds it.

## Make the multipliers visible

- Average concurrent work is approximately arrival rate × average residence time in a stable system. At an assumed 2,000 requests/s and 50ms residence, that is 100 requests, before bursts and variance. This is not a p99 capacity formula.
- In-flight memory includes simultaneous encoded input, decoded representation, working set, output and queued references. Multiply by admitted concurrency and include shared/native/page-cache memory separately.
- Storage growth includes authority bytes + indexes + projections + retained versions/events + WAL/AOF + replicas + backups + temporary rebuild space. Report physical compression/deduplication separately from logical payload size.
- Fan-out and retry multiply dependency demand. Three attempts at each of several layers are not three attempts in total. A slow dependency increases both residence time and retained memory.
- A cache's useful value depends on reuse, skew, hit cost, coherence and cold-miss load. A percentage hit rate alone does not show which expensive requests remain.

These are accounting models, not universal sizing coefficients. Distinguish CPU service demand from wall-clock waiting. Examine utilization, queued demand and errors per constrained resource using the [USE method](https://www.brendangregg.com/usemethod.html).

## Design with headroom and failure load

Specify useful throughput and latency by operation class and tenant distribution. Size the surviving system for the intended failure domain, including reconnects, cold caches, retries and catch-up. A steady-state total across all healthy instances is insufficient to establish failover capacity.

Give queues a count/byte limit, admission deadline, priority/fairness policy and overload response. A longer queue trades immediate rejection for memory and waiting; it does not create capacity. Estimate the largest object and decompression expansion, not only average request size.

Before choosing another service, shard or cache, compare the saved cost with new network hops, copies, update traffic and operational state. For validation, use [measurement](../performance/measurement.md): target hardware, data skew, realistic arrivals and explicit pass/fail budgets. Revisit estimates after implementation without retroactively presenting them as benchmarks.
