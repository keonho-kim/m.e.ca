# Product scenarios as design exercises

These are workload-dependent examples, not standard stacks. Derive the authority, access paths and resource costs from the actual product. Read only the relevant linked references.

## Embedded/local application

A single host owns durable state, reads dominate and write bursts fit a bounded queue. SQLite can remove network/server overhead while preserving relational transactions. Keep large immutable content separate from frequently updated metadata, and return summaries without decoding original payloads. Design writer admission, checkpoint progress and consistent backup together. Test simultaneous processes and a reader that survives too long. See [SQLite](../storage/sqlite.md).

The reason to move to a shared server is a concrete multi-host authority, contention, recovery or operational requirement; a file's size or a guessed user count alone is not a sufficient trigger.

## Multi-tenant SaaS and workflow

The expensive paths may be filtered lists, authorization, concurrent approval and retained audit history. Choose tenant-scoped indexes, versioned conditional transitions and a durable command identity. A PostgreSQL authority may satisfy the requirement without an additional cache; add Redis for demonstrable shared hot state with a stated stale/miss contract. Separate audit retention from debug logs. Test the largest tenant alongside small tenants, duplicate submission and revocation. See [transactions](../storage/transactions.md) and [identity/tenancy](../api/identity-and-tenancy.md).

## Commerce or scarce-resource reservation

Inventory, ownership or money requires a database invariant. Define reservation → commitment → release/expiry, with conditional transitions and command lookup after timeouts. External payment/refund calls need their own identity and reconciliation; a Redis lease or queue acknowledgement is insufficient. Use an outbox for post-commit publication and bound unresolved reservations. Test a commit with a lost response, expired workers and repeated external callbacks. See [commands](../api/commands-and-idempotency.md).

## Search/knowledge/AI service

Original content and permission state remain authoritative; text/vector indexes and result caches are rebuildable projections. Narrow candidates before hydrating large content, preserve access checks for snippets/facets, and budget embedding/reranking work independently from API concurrency. Version model, tokenizer and index generations. Measure quality under restrictive filters, update lag and deletes, not only vector-query speed. See [retrieval](../performance/vector-retrieval.md).

## Collaboration or long-running generation

Choose whether the server owns work across disconnects. Persist only the state/events required for recovery and replay; coalesce disposable progress updates. A slow subscriber needs a bounded queue and reconnection/resync path. Avoid one database transaction, complete payload copy or goroutine per retained history item. See [streams](../api/streams.md).

## Analytics, import/export and scheduled processing

Separate interactive capacity from long scans, decompression and batch writes. Estimate sorting, joins, spill space and downstream pressure; prefer bounded batches and resumable checkpoints over loading all rows. An append-only analytical projection or specialized engine earns its cost only when column selection, aggregation or isolation benefits exceed ingestion and operating costs. See [query paths](../performance/query-paths.md) and [jobs](../api/jobs-and-webhooks.md).
