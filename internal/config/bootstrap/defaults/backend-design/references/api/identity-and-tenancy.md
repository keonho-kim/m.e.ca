# Identity, tenancy and authorization cost

Use when a design shares pools, caches, indexes or workers across principals. Optimize repeated checks without weakening their scope or freshness.

## Establish trusted identity once

Resolve authentication at the ingress boundary, then propagate a validated caller/tenant capability through the use case. Do not trust a tenant identifier merely because it appears in a header, cursor or job payload. Perform object/action authorization at the actual read/effect, including background execution, detail downloads and replay.

For OAuth deployments, apply the current [security BCP](https://www.rfc-editor.org/rfc/rfc9700.html) to the actual client flow. For JWTs, pin accepted algorithms and validate issuer, audience, purpose and lifetime; fetch verification keys only from trusted configured locations. Avoid using an unverified token URL/key field for network access. See [JWT BCP](https://www.rfc-editor.org/rfc/rfc8725.html).

Cache verified signing keys with bounded refresh and planned rotation overlap; unknown-key floods must not trigger unbounded synchronous fetches. Separate cryptographic signature validity from token revocation and object authorization. A short token lifetime bounds some exposure but is not immediate revocation.

Sender-constrained credentials such as [DPoP](https://www.rfc-editor.org/rfc/rfc9449.html) can reduce stolen-token replay when client/server support the protocol. Include proof validation and replay state cost; they do not protect a client whose authorized execution is compromised.

## Scope every shared optimization

Bind cache keys, deduplication, batching, object references and cursors to the relevant tenant/visibility context. Never let cross-tenant coalescing return a result computed under another principal's permissions. Authorization decisions may be cached only with a defined version/freshness/revocation contract.

Database RLS can add defense in depth, but table owners, superusers and `BYPASSRLS` roles have special behavior. Verify `USING`/`WITH CHECK`, privileged maintenance paths and actual service roles. PostgreSQL's [RLS documentation](https://www.postgresql.org/docs/current/ddl-rowsecurity.html) describes those boundaries. Application-supplied session context is not isolation against an actor who can execute arbitrary SQL as that role.

With pooled connections, set scoped state transaction-locally and prove cleanup across success, rollback and cancellation. An admin connection must not leak into tenant work. Composite tenant/object keys and foreign keys can enforce relationships that a handler-only check misses.

## Freshness and resource isolation

Permission revocation plus newer content requires an ordering guarantee when the product promises immediate exclusion. The [Zanzibar paper](https://storage.googleapis.com/gweb-research2023-media/pubtools/5068.pdf) demonstrates the stale-permission/new-content problem; use a simpler local version/transaction mechanism when it meets the same need.

Apply per-tenant admission and cost-weighted limits alongside global limits. A tenant with expensive filters or huge objects can consume more capacity at the same request count. Do not put unbounded tenant/user IDs into metric labels; use bounded cohorts and authorized diagnostic detail.

Test two tenants with identical object IDs, cached denial/allow after revocation, pooled-state leakage, retry after identity change, deleted objects in search snippets, and privileged exports. An empty result or approximate retrieval shortcut must not hide a missed authorization check.
