# File sharing Do & Don’t

| Decision | Do | Don’t |
| --- | --- | --- |
| Transfer and readiness | Track the actual phase. Confirm any required processing before offering the ready resource. | Report a file available from transfer percentage alone. |
| Associations | Store the attachment's scoped owner and stable ID. Verify the parent and audience when reading it. | Put an unrestricted storage path into a private message or document. |
| Validation | Validate service-owned limits and file completeness before accepting it. | Depend solely on the browser's filename extension or input hints. |
| Batch retry | Show item-level results and continue with explicitly retryable items. | Restart successful transfers or reattach them blindly after a timeout. |
| Sharing | Define expiration, revocation and the usable authorization point for original and preview. | Promise instant revocation while an already issued link can keep serving bytes. |
| Cleanup | Expire abandoned attempts and release previews according to the actual ownership and retention rules. | Delete a shared file because it disappeared from one view without checking other references. |
