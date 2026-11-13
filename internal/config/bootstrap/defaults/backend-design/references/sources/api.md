# API and execution primary sources

Protocol standards, provider policies and deployment research have different scopes. Follow the [API map](../api.md) for the design synthesis.

## Protocol and description

| Source | Status/scope and useful decision |
| --- | --- |
| [RFC 9110](https://www.rfc-editor.org/rfc/rfc9110.html) | HTTP semantics; methods, preconditions and representation behavior |
| [RFC 9457](https://www.rfc-editor.org/rfc/rfc9457.html) | Standards Track HTTP problem details; does not define business retry policy |
| [OpenAPI](https://spec.openapis.org/oas/latest.html) | Latest resolved to 3.2.1; validate actual generator/validator support |
| [SSE](https://html.spec.whatwg.org/multipage/server-sent-events.html) | HTML Living Standard framing, reconnection and `Last-Event-ID` |
| [RFC 6455](https://www.rfc-editor.org/rfc/rfc6455.html) | WebSocket protocol and origin/framing model |
| [gRPC deadlines](https://grpc.io/docs/guides/deadlines/) | Explicit deadlines, propagation and server work cancellation |
| [gRPC flow control](https://grpc.io/docs/guides/flow-control/) | Stream buffering and bidirectional deadlock considerations |
| [gRPC retry](https://grpc.io/docs/guides/retry/) | Attempt configuration, commitment and retry observation |

## Identity and authorization

| Source | Status/scope and useful decision |
| --- | --- |
| [RFC 9700](https://www.rfc-editor.org/rfc/rfc9700.html) | OAuth 2.0 Security Best Current Practice, January 2025 |
| [RFC 8725](https://www.rfc-editor.org/rfc/rfc8725.html) | JWT Best Current Practice; algorithm, context and claim validation |
| [RFC 9449](https://www.rfc-editor.org/rfc/rfc9449.html) | Standards Track DPoP token sender constraint and replay model |
| [Zanzibar publication](https://research.google/pubs/zanzibar-googles-consistent-global-authorization-system/) and [paper](https://storage.googleapis.com/gweb-research2023-media/pubtools/5068.pdf) | Deployed Google authorization research, 2019; causal permission/content ordering |

## Durable execution and external boundaries

| Source | Scope and useful decision |
| --- | --- |
| [Stripe idempotency API](https://docs.stripe.com/api/idempotent_requests) | Actual provider semantics, including API-generation differences |
| [Stripe API design account](https://stripe.com/blog/idempotency) | Motivation for stable command identity under network failures |
| [Stripe webhooks](https://docs.stripe.com/webhooks) | Provider signature, replay, ordering and duplicate-delivery handling |
| [Debezium outbox router](https://debezium.io/documentation/reference/stable/transformations/outbox-event-router.html) | Concrete transactional-publication adapter and record model |
| [Kafka 4.1 design](https://kafka.apache.org/41/design/design/) | Delivery semantics and scope of transactional processing |
| [S3 presigned URLs](https://docs.aws.amazon.com/AmazonS3/latest/userguide/using-presigned-url.html) | Bearer capability scope, reuse, integrity and overwrite semantics |
| [OWASP file uploads](https://cheatsheetseries.owasp.org/cheatsheets/File_Upload_Cheat_Sheet.html) | Format, storage and expansion/security boundaries |
| [OWASP SSRF](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html) | Resolved destination/redirect and network-boundary considerations |

OWASP sheets are maintained security guidance, not wire standards. Stripe's retention and retry rules belong to Stripe APIs, not every application. A queue's delivery semantics do not by themselves establish the atomicity of an arbitrary external effect.
