# API contracts with bounded execution cost

Design the external operation and its internal work together. Specify scope, accepted bytes/items, filtering/sort semantics, consistency, preconditions, cancellation and outcome. OpenAPI/protobuf describe the wire contract; they do not replace the use case's effect model. Select tooling that actually supports the chosen [OpenAPI version](https://spec.openapis.org/oas/latest.html).

## Reads and lists

Return a projection appropriate for the view rather than a full domain aggregate plus every relationship. Scope authorization and filtering before pagination; bind opaque cursors to query, tenant, ordering and version. Use a stable unique tie-breaker. Keyset pagination reduces discarded rows, but changing sort values and membership still require a live/snapshot contract. See [query paths](../performance/query-paths.md).

Separate list metadata from expensive detail or large original bytes. Batch related lookups where it saves round trips, while bounding IDs and output size. Avoid N+1 calls hidden by an ORM, GraphQL resolver or serialization hook. A request-local loader should not become an unbounded cross-tenant global cache.

Conditional GET and representation validators can avoid transfer and serialization; `If-Match` can express update preconditions. Validate at the authoritative write, not against a stale cached copy. Cache identity and `Vary` must preserve caller visibility. See [HTTP semantics](https://www.rfc-editor.org/rfc/rfc9110.html).

## Mutations and bulk work

Parse one unambiguous input before starting effects. Reject oversized/deep/ambiguous structures at the boundary, with limits on decompressed bytes as well as encoded input. Treat dynamic fields/sorts as allowlisted domain values; parameterization does not make SQL identifiers safe.

Define whether a bulk request is atomic, per-item transactional or asynchronous. Bound count and bytes, preserve result identity/order where promised, and specify partial failure and retry keys. One huge transaction may improve throughput while causing long locks, memory spikes and unfairness. A batch should yield at its designed boundary.

For long-running operations, return durable acceptance only after acceptance is committed. A status resource needs tenant authorization, bounded retention and distinguishable accepted/running/terminal/unknown states. Do not mark success because a worker was queued.

## Errors and protocols

Use registered machine identity and safe messages from the [central diagnostics module](../operations/diagnostics.md). An HTTP adapter can use [Problem Details](https://www.rfc-editor.org/rfc/rfc9457.html); a gRPC adapter uses appropriate status/details. Keep the same operation semantics without wrapping every protocol in an identical ad hoc envelope.

Map conflict/precondition, denial, overload, cancellation, dependency failure and uncertain effect deliberately. HTTP status alone is not a safe retry instruction. Preserve cancellation causes internally, avoid stack/SQL/credential leaks, and emit the terminal diagnostic once at the owning boundary.

Check the contract under maximum page/batch size, a canceled client, concurrent update, denied tenant, stale cursor and partial bulk failure. Verify database calls and encoded bytes per operation as well as response shape; schema conformance alone cannot reveal work amplification.
