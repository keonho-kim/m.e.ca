# Commands, idempotency and uncertain outcomes

Use for any accepted mutation that a caller, proxy or worker might retry. Define the logical command's identity and outcome lookup before enabling retry. An idempotent transport verb does not make a multi-step external effect safe.

## Persist identity with the effect

Bind a key to authenticated scope, operation and a well-defined request fingerprint. Derive the fingerprint from validated semantic input with deterministic representation; exclude irrelevant transport fields and preserve meaningful ordering/precision. If input includes secrets, use an appropriate protected fingerprint and retain no plaintext merely for deduplication.

Enforce uniqueness in the authority. Persist the command reservation, relevant mutation and outcome/operation reference atomically where possible. A repeated identical command returns the established outcome; reuse with changed input is a conflict. Concurrent duplicates need an explicit wait/status/conflict policy and bounded waiters.

Stripe documents concrete [idempotency behavior](https://docs.stripe.com/api/idempotent_requests), including differing API-generation policies. Treat that as a case-specific contract; choose and document the behavior appropriate to this service rather than copying one provider's lifetime or error caching rules.

## Model the uncertain interval

Use states that distinguish unstarted, owned/running, committed, failed with known absence of effects, and unresolved outcome. The exact states depend on the workflow; avoid a universal boolean `success` that loses commit uncertainty.

If the connection disappears after commit, the client must be able to query/retry using the same identity without executing the effect again. If a transaction returns an ambiguous network error, consult durable identity/state before reclaiming the slot or rerunning work. Timeout means observation ended, not that the mutation did not happen.

An external call needs provider-supported identity or a reconciliation protocol. Commit intent/outbox first when appropriate, keep the external request ID, then record the observed outcome. Do not hold a transaction open around a long payment/email/AI call to simulate atomicity. Compensation is another fallible operation with its own identity, not an automatic inverse.

## Bound cost without weakening the promise

Store a compact result or stable result reference when retaining whole responses would dominate storage. Bind reference authorization and version; an idempotent replay should not silently return somebody else's evolving object. Bound payload, attempts, concurrent duplicates, retained records and recovery work.

Set retention from the retry/reconciliation horizon. Once identity is forgotten, the system can no longer claim indefinite deduplication. High-risk business invariants may need a permanent business key even when transport deduplication expires. Cleanup must not remove live/uncertain ownership merely because a request deadline elapsed.

Place retry policy at the layer that understands effect safety and the total deadline. Include SDK/proxy retries in the budget. Amazon's [retry guidance](https://aws.amazon.com/builders-library/timeouts-retries-and-backoff-with-jitter/) explains load amplification and backoff; it does not establish application idempotency.

Verify simultaneous duplicates, changed request body, response loss after commit, external timeout, owner restart, stale lease and cleanup at the retention boundary. Assert one business effect, correct outcome lookup and bounded resource retention, not only identical HTTP responses.
