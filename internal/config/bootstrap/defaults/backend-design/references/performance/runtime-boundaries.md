# Runtime-specific execution boundaries

Read only the target runtime's section. Architecture diagrams must map to its real scheduling, memory and cancellation behavior. Native concurrency is not automatically resource admission or effect cancellation.

## Go

Use a request context and explicit goroutine ownership/join. Channels need count/byte accounting outside their item capacity when items vary. A goroutine blocked on a send can retain its payload indefinitely. Clear references in retained slices/maps and cap reusable buffers; `sync.Pool` is opportunistic reuse, not a guaranteed cache or memory ceiling.

Compare allocation rate, live heap and native/mapped memory separately. A small subslice may keep a large backing array alive. Use [Go diagnostics](https://go.dev/doc/diagnostics) and the [GC guide](https://go.dev/doc/gc-guide) for evidence. Verify driver/context cancellation and response-body closure; returning from a caller does not join unrelated work.

## Node.js

Keep the event loop available for orchestration. Large synchronous JSON/regex/crypto or loops can stall unrelated requests even when APIs around them are async. Partition work or use a bounded worker pool where cost justifies transfer/serialization. [Node's event-loop guide](https://nodejs.org/en/learn/asynchronous-work/dont-block-the-event-loop) explains the boundary; use the actual runtime's [worker API](https://nodejs.org/api/worker_threads.html).

Budget copied/transferred/shared buffers and per-worker memory. A Promise limit is useful only if it bounds the real work before allocation/start. Streams need backpressure and cleanup on destroy/error. Aborting a fetch/wait does not automatically cancel arbitrary worker computation or a remote committed command.

## JVM

Distinguish platform threads, virtual threads, CPU parallelism, database connections and native/direct-buffer memory. Cheap waiting threads do not create more database or CPU capacity. Bound those scarce resources directly; use the runtime/framework's actual cancellation/interrupt contracts and verify blocking library behavior.

Use [Oracle's virtual-thread guide](https://docs.oracle.com/en/java/javase/25/core/virtual-threads.html) for the selected JDK, rather than carrying old release-specific pinning assumptions forward. Avoid retaining large request data in long-lived thread-local/context state. Measure heap, direct buffers, GC and scheduling separately. Do not add a second executor around an existing bounded execution facility without a current isolation need.

## Rust with an async runtime

An async task must yield; blocking computation/I/O on executor threads can delay unrelated futures. Use a bounded dedicated facility for CPU/blocking work, with explicit result/lifetime ownership. Tokio documents that started [`spawn_blocking` tasks](https://docs.rs/tokio/latest/tokio/task/fn.spawn_blocking.html) cannot generally be aborted through the handle; a shutdown timeout does not make them stop.

Keep admission held while the actual blocking work remains live. Add cooperative cancellation or a process boundary when interruption is required. Arc/shared buffers can retain large allocations just as managed-runtime references do. Unsafe/FFI zero-copy paths need validated lifetimes and native cleanup; Rust's type safety does not make a remote transaction cancelable.

## Shared acceptance condition

Test cancellation, saturated pools, slow consumers, unusually large payloads and idle-after-load. Verify actual worker completion and resource release, not just that a future/context returned. Runtime selection changes implementation costs; preserve the operation's same correctness/freshness/quality contract when comparing performance.
