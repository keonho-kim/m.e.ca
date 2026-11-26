# External adapters and ordered wrappers

Choose when a use case depends on an external API, storage engine or identity service. Keep the application's required operation narrow and the remote format/protocol inside an adapter. Select a concrete adapter explicitly at construction or a validated dispatch point; do not introduce Strategy policy-object hierarchies.

## Concrete arrangement

```text
root constructs bounded client/pool and concrete adapter
use case → narrow operation(ctx, identity, input)
  → admission/deadline boundary → adapter → remote service
  ← observed outcome / registered fault with preserved cause
```

The use case owns effect safety and total-operation retry policy. The adapter owns framing, credentials, transport limits, parsing and transport observations. The root owns connection pools, worker lifecycle and shutdown. A proxy must make remote latency, partial effects and failure visible in its contract.

## Compose behavior only where it belongs

| Mechanism | Real use | Ordering consequence |
| --- | --- | --- |
| Decorator/interceptor | Timing or a real cross-cutting boundary | Decide whether it measures one attempt or the whole operation |
| Authorization proxy | Guard access at a concrete resource boundary | Check before access; avoid cached permission bypass |
| Chain of Responsibility | Several actual resolvers with precedence | Distinguish unhandled from failed; never fall through after uncertain effects |
| Bulkhead | Protect one dependency/work class from another | Bound queue/slots and reserve shared resources consistently |
| Circuit breaker | Repeated dependency failures consume capacity | Count the relevant failures, bound probes and avoid retry amplification |
| Factory/compatible family | Resource construction or paired protocol components | Verify initialization once; fail before publishing a partial family |

A possible order is parse/authenticate → overall budget/admission → permitted attempts → transport encode/send/decode → domain outcome → one terminal diagnostic. Derive the exact order from what is protected. Per-attempt metrics can be inner wrappers while final operation metrics remain outer; do not count both as completed business operations.

Keep retry decision outside generic error formatting. A timeout after sending a command is not interchangeable with a connect failure before sending it. Include client SDK and proxy retries in the total. A failed mutating resolver cannot simply delegate to a second provider: it may duplicate a committed effect.

## Contract variation and cost

Bridge/Component composition fits only when two dimensions really vary independently—for example required output formats and destinations with independent contracts. Prefer direct functions for a few cases. Inheritance hooks such as Template Method/Factory Method may fit an existing framework; do not create a framework merely to name the pattern.

Validate capability differences explicitly: pagination, ordering, cancellation, limits and idempotency cannot be assumed equal behind one interface. Share immutable connection configuration, not request headers or mutable bodies across tenants. Reuse clients and avoid serializing through each wrapper.

Review cumulative timeouts, duplicate copies, hidden goroutines, queue memory and connection counts. Test denied access, body limits, remote partial response, lost acknowledgement, open-circuit probes and shutdown during a blocked call. If one direct adapter call expresses the feature, keep it. See [resilience](../operations/resilience.md) and [integration boundaries](../api/files-and-integrations.md).
