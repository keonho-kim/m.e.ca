# AI workspace Do & Don’t

| Decision | Do | Don’t |
| --- | --- | --- |
| Run identity | Give chat history, run state and tool results stable identities. Reconcile reconnects against confirmed history. | Create a new run just to recover a dropped display stream. |
| Tool authority | Verify the actor, target and permission before executing a consequential operation. | Let retrieved content, a generated message or a stale proposal grant access. |
| Streaming | Render partial output with its actual run scope and preserve useful progress. | Declare completion when a token, heartbeat or connection ends. |
| Cancel | Track the stop request and the outcome of owned work. Show already completed effects. | Promise rollback from dismissing a loading view. |
| Retry | Decide whether the user wants another answer or eligible recovery of an operation. | Replay uncertain tool effects when retrying generation. |
| History | Retain authorized sources and the minimum material needed for recovery. | Expose hidden credentials or another workspace's artifacts in logs or references. |
