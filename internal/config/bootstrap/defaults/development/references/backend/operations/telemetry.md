# Telemetry with a resource budget

Use while choosing the evidence required to validate design, capacity and incidents. Logs, metrics, traces, profiles and audit records answer different questions and should not duplicate complete request bodies.

## Correlate without copying payloads

Use stable request/operation/job identity with trace/span correlation where available. Separate occurrence and observation times when delayed delivery matters. The [OTel log data model](https://opentelemetry.io/docs/specs/otel/logs/data-model/) supplies common fields; [W3C Trace Context](https://www.w3.org/TR/trace-context/) standardizes propagation, not caller authentication.

Treat incoming trace context/baggage as untrusted bounded metadata. Do not put credentials, personal data or unrestricted client input into propagated attributes. At an external boundary, decide which context to accept/forward and whether to start a new trust-scoped trace. Follow [OTel sensitive-data guidance](https://opentelemetry.io/docs/security/handling-sensitive-data/) before exporting to another service.

## Instrument the queue and the effect

Measure arrivals, admissions, rejections, attempts and useful completions separately. Decompose waiting from service work: pool wait, lock wait, query, external dependency, serialization and client write. For jobs/streams, include queue age, retained bytes, oldest unacknowledged item, first useful output and terminal outcome.

Choose a bounded label vocabulary such as operation, result category and deployment/cell. User IDs, arbitrary paths, document IDs and error messages can create unbounded time series; hashing them does not reduce cardinality. Estimate the product of label values, instances and histogram buckets before rollout. See [Prometheus instrumentation](https://prometheus.io/docs/practices/instrumentation/).

Use distributions suitable for fleet aggregation and the actual SLO boundary. Summaries/quantiles cannot simply be averaged into a global percentile. Validate histogram resolution, native-histogram support and query behavior against the [histogram guide](https://prometheus.io/docs/practices/histograms/). Keep client-observed latency distinct from successful server-only latency.

## Sampling and overhead are design choices

Head sampling bounds ingestion early but may miss rare slow/error traces. Tail sampling can retain selected completed traces but needs buffering, routing/co-location, bounded decision time and a policy for incomplete traces. Neither should become the sole durable audit record.

Bound exporter queues, batches, attribute count/length and retries. Decide what is dropped and expose that loss. Full-fidelity SQL values, request bodies and high-rate per-item events may cost more than the operation being observed. Prefer aggregate counts and a targeted bounded diagnostic window when it answers the question.

## Alert on an actionable objective

Define the service's eligible/good events and latency/error objectives before choosing thresholds. Multi-window burn-rate alerts can connect sustained error-budget consumption to action; [Google's SLO alerting chapter](https://sre.google/workbook/alerting-on-slos/) explains tradeoffs. Queue age, disk/key expiry and lost telemetry may need separate predictive alerts.

Do not mark a successfully handled conflict/retry as a final application failure merely because a lower-level attempt failed. Preserve attempts as dependency evidence and classify the complete operation separately. Register event/error policy centrally in [diagnostics](diagnostics.md).

Verify dashboards against intentionally failed, rejected, canceled and retried operations; confirm no double counting and no secrets. Measure telemetry CPU, allocation, network, disk and outage behavior as part of the service's own resource budget.
