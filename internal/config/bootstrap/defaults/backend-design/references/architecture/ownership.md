# Ownership that avoids unnecessary work

Use when designing modules or untangling handlers, persistence, execution and diagnostics. Split at a decision, state or I/O boundary that has a real consumer. A smaller file alone does not reduce coupling or runtime cost.

The [feature-module composition](../compositions/service-module.md) provides an adaptable source arrangement and call/dependency path. Use the table below to assess ownership; do not instantiate one package or interface for every row.

## Assign the actual authority

| Concern | Owns | Should receive |
| --- | --- | --- |
| Transport adapter | Parsing, framing, transport limits and outcome mapping | A narrow application operation |
| Application/use case | Authorization-dependent orchestration, effect ordering and cancellation | Parsed domain values and explicit I/O collaborators |
| Domain feature | Business invariants and state transition rules | Only the facts needed for its decision |
| Persistence adapter | Queries, transactions, conditional writes and stored representation | An operation expressing the required atomicity |
| Diagnostics module | Codes, templates, event/field definitions and publication policy | Registered definitions, bounded fields and causes |
| Composition root | Concrete construction, shared pools and shutdown | Configuration and owned resource handles |

Keep transaction ownership visible. Splitting two writes behind separate repositories does not preserve atomicity unless they share a transaction deliberately. Prefer a cohesive operation to an interface that leaks a database transaction through every layer.

## Avoid structural performance taxes

Map untrusted input once at the boundary, then pass typed values. Avoid converting one request through maps, JSON and transport DTOs between in-process modules. Do not allocate a second complete history, tree or blob merely to cross a package boundary. Immutable views can share backing storage only when their lifetime and mutation rules are explicit.

Build shared pools, clients and log sinks once where their lifetimes require it; keep request state local. Lazy allocation is useful for optional expensive data, but initialization failure, cancellation and concurrent first use still need an owner. A cache added to hide duplicate computation can be inferior to removing the duplicate owner.

Separate pure decisions from external effects so a retry can recompute safely without repeating publication. If a command commits and a notification fails, return or persist that partial outcome; do not let a generic error handler rerun the whole use case. [HTTP idempotency semantics](https://www.rfc-editor.org/rfc/rfc9110.html) do not make arbitrary application effects idempotent.

Central diagnostics is a dependency leaf: it does not import domain, HTTP or database packages. Domain and adapter errors carry registered identity and safe structured context; adapters map to their own transport types. Infrastructure causes remain inspectable without becoming public error text. See [central diagnostics](../operations/diagnostics.md).

## Review the boundary

Trace one success, one denied operation, one timeout after commit and one shutdown. Confirm who releases rows, buffers, streams, leases and background work. Verify that the split adds no repeated query, serialization, deep copy, global lock or hidden goroutine. Benchmark a hot boundary only when it actually changes its path; use behavior and dependency review for a purely structural move.
