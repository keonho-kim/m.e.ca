# Collaborative documents Do & Don’t

| Decision | Do | Don’t |
| --- | --- | --- |
| Save | Carry a base revision or equivalent edit identity and track acknowledgements. | Claim saved because a local buffer or autosave timer updated. |
| Conflicts | Preserve the person's unsaved edits and give a useful comparison or retry. | Resolve conflict by silently overwriting the other revision. |
| Restore | Specify what restoring changes, including comments and published or shared output if present. | Apply a historical draft to every destination by default. |
| Comment anchors | Keep enough revision or content context to resolve and reattach a comment. | Attach feedback to an unrelated paragraph after an edit changes positions. |
| Permissions | Recheck at the write, history and export boundaries. | Assume a viewer's initially loaded permissions are still valid. |
| Offline state | Keep a scoped recoverable draft where the product supports local storage. | Present a local draft as the latest shared version. |
