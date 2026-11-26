# Offline fieldwork Do & Don’t

| Decision | Do | Don’t |
| --- | --- | --- |
| Status | Label local save, queued send, acknowledged sync and conflict by their actual state. | Call a locally finalized record synced. |
| Intent identity | Reuse the original operation identity and map the accepted record durably. | Create a new remote record for each reconnect. |
| Scope | Bind local records, cached data and sync requests to their actor and workspace. | Replay one account's queue under another account after login changes. |
| Verification | Apply current server authorization, schema and required-file checks on acceptance. | Trust an old local permission or form version to bypass current acceptance rules. |
| Conflict | Keep the useful local work and expose the actual cause and recovery path. | Overwrite conflicting changes based only on the device clock. |
| Storage | Bound local retention and cleanup according to the feature and data's sensitivity. | Leave sensitive drafts on a shared device without an explicit ownership and retention policy. |
