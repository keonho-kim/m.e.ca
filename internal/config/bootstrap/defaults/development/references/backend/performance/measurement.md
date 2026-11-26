# Performance evidence from design to production

For a new design, state a resource-cost hypothesis and acceptance budget. For existing code, capture a reproducible baseline. Use measurements to test the chosen explanation; do not require a running system before reasoning about data movement and asymptotic work.

Use [design studies](../design-studies.md) for explicit assumed-cost comparisons and [invariant verification](../operations/invariant-verification.md) to ensure a faster result still fulfills the same state/effect contract.

## Select evidence for the question

| Question | Useful evidence |
| --- | --- |
| CPU work or repeated parsing? | CPU profile, allocation profile, work counts |
| Waiting on locks/pools/I/O? | Pool/lock wait, blocking/off-CPU stacks and traces |
| Memory leak or expected cache? | Retained object graph, idle recovery, live/RSS/native trends |
| Query/index design? | Plans, rows/loops, buffer reads, spill and write/WAL bytes |
| Saturation/capacity? | Useful throughput, arrival/admission rate, queue age and p99 |
| Retrieval approximation? | Judged/exact quality baseline sliced by filters/tenants |

CPU sampling cannot explain all wall-clock waiting; [off-CPU analysis](https://www.brendangregg.com/offcpuanalysis.html) complements it. Start from constrained-resource utilization, queued demand and errors rather than one dashboard's CPU percentage.

## Build a representative workload

Fix versions, hardware/container limits, dataset/seed, object-size distribution, tenant skew, read/write mix, connection reuse and durability settings. Separate warm/cold cache and startup/recovery. Include typical and maximum accepted input, deep pages, concurrent writes and maintenance.

For independent external arrivals, an arrival-rate/open model reveals overload that a closed loop may hide by sending less traffic when the server slows. A closed model still fits a genuinely fixed interactive population. Record intended arrivals, actual starts, admissions, rejections and completions. See [k6's model explanation](https://grafana.com/docs/k6/latest/using-k6/scenarios/concepts/open-vs-closed/).

Use enough duration/samples for steady behavior, GC, checkpointing and background work. Report distributions and uncertainty; do not average p99 values into a global p99. Collect latency histograms correctly across instances using the supported [histogram model](https://prometheus.io/docs/practices/histograms/). Match bucket resolution to the objective and account for telemetry overhead.

## Compare equivalent work

Preserve result quality, consistency, authorization, durability and error behavior. Report useful completed work separately from attempts, skipped work and early errors. Track CPU, peak/retained memory, allocation rate, disk/index/WAL bytes, network bytes and cost beside latency. A speedup obtained by disabling fsync, truncating results or omitting checks is a changed contract.

Microbenchmarks explain a hot mechanism; end-to-end tests determine whether it matters. Repeat paired runs with the same workload and inspect variance. When claiming no regression, define the relevant workload and tolerance rather than treating a non-significant small sample as proof of equality.

## Exercise the expensive failure path

Include dependency latency/loss, cache outage, reconnect burst, duplicate command, slow subscriber, canceled large request, full disk and interrupted maintenance where relevant. Confirm queues and retained state return to their expected idle bound. Perform disruptive experiments only in the authorized environment, with isolated data and a stopping condition.

Carry the same budget into rollout: canary representative traffic, monitor error/latency/capacity and define a reversible stop rule. Retain source/revision, commands, inputs and observed results without credentials or live payloads. Synthetic or loopback evidence should be labeled as such; it does not establish real-provider, production-scale or cross-platform performance.
