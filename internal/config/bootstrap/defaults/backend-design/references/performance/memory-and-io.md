# Memory, serialization and I/O ownership

Design the lifetime and representation of bytes before adding pooling or concurrency. Track allocation rate, retained heap, native allocations, mapped files/page cache, socket buffers and temporary disk separately. A low allocation benchmark is not a process-memory bound.

## Prevent multiplicative retention

Trace encoded input → parsed value → domain state → stored payload → output projection → encoded output. Remove complete copies that have no independent mutation/lifetime need. Decode once per relevant ancestor or batch, project selected fields early and avoid parse/encode cycles between in-process modules.

Sharing immutable backing bytes can save copies, but a small slice/view may pin an entire large allocation. Clone a small survivor when it substantially shortens retention. Release request/worker references after completion; unbounded closures, caches and subscriber queues can keep otherwise dead bodies reachable. Go's [GC guide](https://go.dev/doc/gc-guide) explains allocation/retention and soft-limit tradeoffs; runtime limits are not a guarantee against total RSS/OOM.

Canonical persisted payloads can avoid repeated database blobs. Keep authenticated references small and budget restoration separately from metadata paging. Streaming a reconstructed original still may require a large first decode; do not advertise constant-memory behavior without inspecting the actual format and path.

## Choose concurrency and buffers together

Bound work by cost as well as count: compressed images, large JSON, encryption and reranking have different expansion/CPU costs. Acquire before allocation. Avoid hold-and-grow deadlocks in a shared weighted budget; reserve a known upper bound or fail/release safely. Propagate cancellation through pool waits, I/O and workers, then join them.

Batch by count, bytes and maximum delay. Retain connection reuse without creating one client/pool per request. Close rows, bodies, subscriptions, files and subprocess resources at their actual owner. Slow consumers need byte-limited queues and write deadlines.

Pool only a measured reusable allocation. Cap pooled capacity and discard oversized buffers; clear retained references and sensitive material as the implementation permits. A pool is not ownership, a hard memory limit or guaranteed secret zeroization. Native FFI/zero-copy APIs need explicit lifetimes that survive callbacks and cancellation without use-after-free.

## Escalate from locality evidence

| Evidence | Candidate | Check before adoption |
| --- | --- | --- |
| Repeated small encode/write calls | Bounded buffered/coalesced output | First-event latency and cancellation |
| Large disk/network transfer copies | Supported streaming/sendfile path | TLS/transformation support and lifetime |
| Pointer-heavy hot traversal | Compact arrays/IDs or column-oriented layout | Mutation, sparse cases and build cost |
| Compute-heavy scans | Maintained vectorized/SIMD implementation | Hardware portability and exactness |
| Hot stable compiled path | Profile-guided optimization | Representative profile and full workload regression |

Use [diagnostic profiles](https://go.dev/doc/diagnostics) to separate CPU, heap, blocking and scheduling. For compiled Go, [PGO](https://go.dev/doc/pgo) consumes representative CPU profiles; its published gains are not this application's result. Kernel async I/O, custom allocators or lock-free structures need measured benefit large enough to justify portability, cancellation and reclamation complexity.

Validate repeated large/small requests, cancellation, errors, idle-after-load and shutdown. Compare peak/steady retained bytes, allocation/copy rate, GC/CPU, descriptors, temporary files and p99 at equal useful throughput. Include warm-up, backpressure and cleanup in the measurement.
