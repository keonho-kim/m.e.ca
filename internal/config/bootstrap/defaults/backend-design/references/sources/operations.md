# Operations and diagnostics primary sources

Company accounts document their own production experience; standards/specifications describe narrower data or protocol contracts. Use the [operations map](../operations.md) for design guidance.

## Production reliability accounts

| Source | Scope and useful decision |
| --- | --- |
| [Amazon: timeouts/retries/jitter](https://aws.amazon.com/builders-library/timeouts-retries-and-backoff-with-jitter/) | Dependency budgets and retry amplification |
| [Amazon: load shedding](https://aws.amazon.com/builders-library/using-load-shedding-to-avoid-overload/) | Protecting useful serving work under excess demand |
| [Amazon: avoiding fallback](https://aws.amazon.com/builders-library/avoiding-fallback-in-distributed-systems/) | Untested fallback and correlated overload costs |
| [Amazon: static stability](https://aws.amazon.com/builders-library/static-stability-using-availability-zones/) | Pre-existing capacity and impaired-control-plane survival |
| [Google: handling overload](https://sre.google/sre-book/handling-overload/) | Per-customer/resource protection and graceful degradation |
| [Google: cascading failures](https://sre.google/sre-book/addressing-cascading-failures/) | Retries, health checks, restart and load-shift feedback |
| [Google: SLO alerts](https://sre.google/workbook/alerting-on-slos/) | Actionable error-budget/burn-rate policies |
| [Google: canary releases](https://sre.google/workbook/canarying-releases/) | Representative evaluation, controls and shared failure domains |
| [Stripe: online migrations](https://stripe.com/blog/online-migrations) | Staged backfill, cutover and removal in a deployed service |

Amazon engineering accounts describe specific production workloads. Confirm that their assumptions and configurations apply to the target service.

## Diagnostic and telemetry contracts

| Source | Scope and useful decision |
| --- | --- |
| [OpenTelemetry log data model](https://opentelemetry.io/docs/specs/otel/logs/data-model/) | Event/observation time, severity, trace and structured fields |
| [OpenTelemetry recording errors](https://opentelemetry.io/docs/specs/semconv/general/recording-errors/) | Handled/propagated errors and avoiding duplicate exception records |
| [OpenTelemetry sensitive data](https://opentelemetry.io/docs/security/handling-sensitive-data/) | Collection/redaction/export trust boundaries |
| [W3C Trace Context](https://www.w3.org/TR/trace-context/) | Recommendation for interoperable trace propagation |
| [Prometheus instrumentation](https://prometheus.io/docs/practices/instrumentation/) | Label cardinality and application evidence |
| [Prometheus histograms](https://prometheus.io/docs/practices/histograms/) | Fleet distributions, aggregation and implementation support |

Check the maturity and version of each OTel convention independently before adopting it. The catalog ownership rule is an application architecture decision, not an OTel requirement.

[The Tail at Scale publication record](https://research.google/pubs/the-tail-at-scale/) identifies Dean/Barroso's 2013 paper. Consult the paper's workload and assumptions before applying its mechanisms; publication metadata alone does not support a numeric speedup or deployment recipe.
