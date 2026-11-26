# Study: encrypted document storage and key service

Fictional inputs: documents average 8MiB; at most 32 active document operations are proposed. The system must restore retained backups and survive interruption during key rotation. The threat includes a stolen database/object-store dump without key-service permission.

## Composition and authority

```text
document use case → scoped metadata/permission lookup
  → bounded key-service adapter → maintained streaming AEAD
  → private immutable object → verified publication → manifest commit
manifest: tenant/object/version, algorithm/key identity, wrapped DEK, object reference
rotation worker: bounded rewrap/re-encrypt progress with recovery state
```

Use a [bounded pipeline](../compositions/pipeline.md) for document bytes, an [external adapter](../compositions/integrations.md) for KMS and a transactional manifest operation. Authentication data binds trusted tenant/object/format context. No stage logs plaintext keys, content or signed URLs.

## Compare key scope and reuse

| Choice | Potential benefit | Cost/tradeoff |
| --- | --- | --- |
| DEK per object/version | Narrow data-key exposure and erasure scope | Wrapped metadata and key-service/cache entries per object |
| Bounded tenant/epoch DEK | Amortized setup and fewer distinct keys | Broader exposure, usage/nonce limits and coarser erasure |
| Bounded plaintext DEK cache | Reduce repeated unwrap latency | Sensitive retained memory, revocation policy and accounting |

Choose from threat/lifetime, not solely KMS throughput. With an assumed 400 reads/s, one unwrap per cache miss and a hypothetical 95% hit rate, unwrap demand would be 20/s before retries/maintenance; this is a model, not a promised hit rate. Include cold start, rotation and cache outage in capacity. The [key lifecycle guide](../security/key-lifecycle.md) owns limits, recovery and retirement.

## Account for bytes and failure

If each of 32 operations simultaneously retained three full 8MiB representations, that would be 768MiB of payload buffers. Two 256KiB processing buffers per operation would total 16MiB for those buffers, but framing, crypto, SDK, network and runtime allocations remain. A streaming API is only a memory improvement if the actual codec/adapter avoids whole-object buffering.

Publish the manifest only when its referenced ciphertext/key information is durable and valid. If object upload succeeds but manifest commit is uncertain, reconcile by stable operation/object identity before deletion or replay. Keep orphan cleanup distinct from a live/unknown publication.

Rotation can change KMS material, rewrap DEKs or re-encrypt data; they have different I/O and exposure effects. Old backups may still depend on old key versions. The worker claims/fences progress, writes a recoverable new envelope, conditionally switches it, and retires old dependencies only when policy permits. Avoid global synchronous rewrites during a read.

## Verification

Test wrong tenant/associated data, invalid/truncated streams, key permission loss, concurrent reads/rotation, failure around object/manifest publication and restore of an old retained backup. Measure KMS calls and latency separately from encryption CPU, payload copies, temporary bytes and total p99. Advanced encrypted computation is unnecessary unless the service itself must not see plaintext; then the product's search/recovery contract changes.
