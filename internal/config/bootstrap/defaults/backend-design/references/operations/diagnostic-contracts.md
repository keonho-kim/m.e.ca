# Concrete central diagnostic contracts

Use with [diagnostics ownership](diagnostics.md) to turn policy into an implementable module. The names below are fictional catalog entries, not mandated product codes or strings to redeclare at call sites.

## Dependency and publication shape

```text
diagnostics/catalog     definitions only; no HTTP/SQL/business imports
diagnostics/fault       registered code + safe fields + inspectable cause
diagnostics/publish     bounded encoding/queue, redaction and sink health
adapter/http|grpc       category-to-transport mapping
application operation  final outcome and retry/reconciliation decision
entrypoint              create sinks/publisher, flush deadline, join
```

These are responsibilities; a small module may keep several in one file. No mutable global service locator or custom logging framework is required.

## An illustrative definition

```text
catalog entry ApprovalVersionConflict:
  code: approval_version_conflict
  category: conflict
  public_message: "The approval changed. Reload its current state."
  event: approval_rejected
  severity: info
  allowed_fields: operation_id, expected_version, observed_version
  disclosure: public message and approved details; cause internal only
```

Only the catalog declares those strings/field identities. A caller supplies `ApprovalVersionConflict` and typed values. It cannot substitute arbitrary text, unknown fields, SQL or a remote body. A third-party error remains an inspectable cause, not a public message. Bounds apply to cause capture/export too; private is not synonymous with safe to persist.

| Catalog category | HTTP example | gRPC example | Operation decision |
| --- | --- | --- | --- |
| Invalid argument | 400 | INVALID_ARGUMENT | Reject before mutation |
| Domain conflict | 409 | ABORTED | Reload/re-evaluate if contract permits |
| Denied | 403, or deliberate concealment policy | PERMISSION_DENIED | Do not retry under the same authority |
| Capacity exhausted | 429 or 503 by scope | RESOURCE_EXHAUSTED | Apply the declared admission/retry contract |
| Unexpected failure | 500 | INTERNAL | Investigate; do not infer absence of effects |

These mappings are illustrative policy. HTTP conditional-request failure may require 412 rather than a generic 409; transport semantics still matter. Retry safety/unknown commit are operation facts, not a static boolean inferred from category/status. [Problem Details](https://www.rfc-editor.org/rfc/rfc9457.html) can carry the registered public identity without exposing its cause.

## One terminal diagnostic

Define which boundary owns completion: request handler for request-owned work, worker/workflow owner for durable work, process root for startup/shutdown. Adapters return faults and may record distinct attempt metrics; they do not all log the same propagated exception. Correlate attempts and final outcome by identifiers without counting each attempt as another business failure.

Check enablement before expensive context extraction. Validate catalog definitions at development/build time and publish only allowed bounded values at runtime. Keep disabled diagnostics cheap, avoid stringify-parse-redact-stringify chains, and separate durable audit intent from best-effort diagnostic emission.

Use compiler/types or focused static checks to detect undeclared codes/events and bypass logger calls where the project supports it; avoid broad suppressions. Verify unknown definitions, secret-bearing causes, duplicate propagation, full queue/disk, exporter recursion and bounded shutdown. Measure enabled and disabled cost using representative structured fields.
