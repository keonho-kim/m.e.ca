# Central logger and error-handler design

Use when defining backend diagnostics or removing scattered logging/error policy. Centralize stable knowledge without moving domain decisions into a catch-all handler.

For the concrete definition shape, dependency direction, transport mapping and caller example, use [diagnostic contracts](diagnostic-contracts.md). This file owns the shared policy; examples illustrate it without introducing a second catalog.

## Give definitions one owner

A dependency-leaf diagnostics module owns error codes, safe message/action templates, event names, severity defaults, public category and allowed structured fields. Other packages construct/emit registered definitions with values and an optional cause; they do not invent codes, event strings or public error messages. Parameter values carry observations, not ad hoc replacement templates.

Keep public text and internal causes distinct. Preserve typed causes/chains for `is/as`-style inspection; do not classify by substring or flatten the cause into the client response. Third-party failures are wrapped into registered operation context at the relevant boundary. Unexpected failures map to one safe internal category and a correlation reference.

| Owner | Responsibility |
| --- | --- |
| Catalog | Stable identities, templates, categories, event/field schemas and disclosure rules |
| Fault value | Registered identity, bounded context and inspectable cause |
| Application operation | Whether it failed/partially committed, cancellation, reconciliation and retry safety |
| HTTP/gRPC adapter | Map the public category/details into that transport |
| Log publisher | Field validation/redaction, encoding, queue and emission policy |
| Composition root | Sink configuration, rotation, lifecycle and bounded shutdown flush |

The catalog must not import business, HTTP, SQL or framework packages. A narrow native error/value API is usually sufficient. Prefer a maintained language logger and existing telemetry integration to a custom formatting/transport framework.

## Avoid duplicated policy and hot-path cost

Register typed/bounded fields rather than arbitrary payload maps. Validate definitions, uniqueness and required fields in development/CI; runtime handling still needs a safe response to invalid values. Keep secret prevention at the earliest capture boundary, then use sink redaction as defense in depth. URLs, SQL parameters, raw exception text, prompts and tokens are not safe by default.

Check enablement before expensive formatting or field extraction. Avoid encoding JSON only to decode it for redaction and re-encode. Reuse immutable catalog definitions, not mutable request field maps. Stack capture is costly; choose it for unexpected failures rather than every expected conflict/cancellation.

Log a propagated failure once where its operation outcome is owned, with useful upstream context. Lower layers may produce distinct bounded observations, but should not log and rethrow the same exception at every boundary. OpenTelemetry's [recording-errors guidance](https://opentelemetry.io/docs/specs/semconv/general/recording-errors/) discusses duplicate/handled exceptions; its current maturity must be checked before freezing conventions.

## Make loss and shutdown explicit

Bound log records, queued bytes/items, file rotation and retained files. Define whether overload drops, samples or blocks each class, and expose dropped/error counters. Exporter failure must not recursively log through the same failing sink. Flush with a deadline and join the owner on shutdown; preserve an inspectable health signal.

Durable audit/business evidence needs its own committed storage/outbox and retention/access contract. A best-effort asynchronous logger is not proof that a command was accepted. Correlate logs with traces using the [OTel log model](https://opentelemetry.io/docs/specs/otel/logs/data-model/); expose safe HTTP errors using [Problem Details](https://www.rfc-editor.org/rfc/rfc9457.html) where appropriate.

Verify cause preservation, transport mapping, denied fields, secret-bearing causes, disabled-log allocations, duplicate emission, full queue/disk, exporter failure and shutdown. Measure throughput and retained bytes with diagnostics enabled and disabled at the intended production configuration.
