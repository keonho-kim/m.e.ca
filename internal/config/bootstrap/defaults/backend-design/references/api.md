# API execution decisions

Classify the effect before choosing HTTP, gRPC or messaging. Transport adapters share the application contract; they should not implement parallel business lifecycles.

Place the use case using the [practical compositions](compositions.md): a direct feature operation, a transactional command, a durable workflow or a bounded pipeline. Keep caller parsing and response mapping at the transport boundary; wrappers must preserve the operation's effect and cancellation semantics.

| Function | Read | Contract that matters |
| --- | --- | --- |
| Lists, detail, mutations, validation and transport schemas | [Contracts](api/contracts.md) | Scope, limits, preconditions and outcome semantics |
| Charges, reservations, submissions and other uncertain mutations | [Commands and idempotency](api/commands-and-idempotency.md) | Durable identity, same-request detection and outcome lookup |
| Progress, generated output, notifications or bidirectional work | [Streaming](api/streams.md) | Ownership, replay, backpressure and terminal state |
| Jobs, schedules, incoming events and outgoing webhooks | [Jobs and webhooks](api/jobs-and-webhooks.md) | Claim/lease, acknowledgement, deduplication and retry ownership |
| Login, authorization, tenant scope and admin access | [Identity and tenancy](api/identity-and-tenancy.md) | Trusted identity and authorization at the actual read/effect |
| Imports, exports, attachments and upstream integrations | [Files and integrations](api/files-and-integrations.md) | Bounded transfer, publication, SSRF and orphan cleanup |

For each operation capture only the meaningful fields: caller/scope, accepted input, effect/commit point, success/partial/unknown outcomes, concurrency rule, cancellation owner, deadline, limits and recovery lookup. A successful HTTP write to a socket does not prove that the caller observed the result.

Choose a command identity before allowing automatic retry. Distinguish delivery acknowledgement, acceptance, durable commit and completed external effects. If work survives the connection, give it a server-owned identity and readable status; if it does not, propagate disconnect cancellation and join owned work.

The [HTTP semantics](https://www.rfc-editor.org/rfc/rfc9110.html) and [OpenAPI specification](https://spec.openapis.org/oas/latest.html) describe interoperable transport behavior. They do not define a product's transaction, authorization, retry or retention policy. Select a schema version supported by the actual validators and generators.
