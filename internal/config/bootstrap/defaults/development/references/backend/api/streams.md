# Streams: ownership, replay and slow consumers

Use for progress, generated responses, notifications or bidirectional operations. Separate work lifetime from subscription lifetime. Decide whether disconnect cancels work or merely stops observation; make cancellation/status explicit when work survives.

## Choose the transport by direction and consumers

| Transport | Good fit | Required checks |
| --- | --- | --- |
| SSE over HTTP | Server-to-client events and browser reconnection | Event framing, replay identity, proxy buffering and authentication |
| WebSocket | Actual bidirectional interaction | Origin/auth, message limits, fair scheduling, heartbeat and flow control |
| gRPC streaming | Typed supported clients and service-to-service streams | Deadlines, message size, flow-control windows and client support |

The [SSE specification](https://html.spec.whatwg.org/multipage/server-sent-events.html) defines event framing and `Last-Event-ID`, not durable execution or unlimited replay. [WebSocket](https://www.rfc-editor.org/rfc/rfc6455.html) defines a bidirectional protocol; product replay and authorization still need design.

## Keep observation bounded

Assign stable operation and event identities where replay is required. Authorize every subscription/resume against current scope. Separate the authoritative outcome from disposable progress. Coalesce replaceable progress updates, but preserve ordered semantic events and terminal state.

Bound subscriber queues by bytes and items. Choose backpressure, selective coalescing or disconnect/resync based on event semantics. One blocked subscriber must not retain an entire execution's payloads or stop unrelated work. Share immutable encoded events only when redaction and visibility are identical; caller-specific projection needs its own boundary.

A replay cursor has an expiry/retention contract. If the cursor predates retained history, return a deliberate resync/snapshot response; do not silently skip missing semantic events. Snapshot plus event tail needs a consistent watermark to avoid gaps/duplicates. Avoid a database transaction held for the life of a client connection.

Use the [snapshot/change protocol](../storage/projection-rebuild.md) to establish that watermark and the [bounded pipeline](../compositions/pipeline.md) for queue, ordering and subscriber ownership. Do not assume event ID allocation order equals commit order.

## Design time and network behavior

Track transport activity, meaningful application progress and total operation deadline separately. A heartbeat can keep a proxy connection alive without proving useful work; it should not indefinitely reset a stalled computation's budget. Select its cadence from product requirements and actual intermediary timeouts, rather than treating one interval as a standard.

Set bounded write deadlines, handle flush errors and stop owned subscription work. Account for compression/proxy buffering when measuring first-event latency. After response headers are sent, a transport cannot replace the response with a normal HTTP error; define terminal event/error and reconnect semantics.

gRPC flow control protects transport buffers but does not bound application queues; manual bidirectional write/read scheduling can deadlock. See [flow control](https://grpc.io/docs/guides/flow-control/) and [deadline propagation](https://grpc.io/docs/guides/deadlines/).

Validate a client reading slowly, a client that stops reading, reconnect before/after terminal state, expired replay, revocation and shutdown during a blocked write. Measure subscriber retained bytes, flush latency, first useful event and recovery traffic. Receiving a heartbeat is not evidence that the requested operation completed or advanced.
