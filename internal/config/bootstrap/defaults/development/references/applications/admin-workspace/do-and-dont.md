# Administrative workspace Do & Don’t

| Decision | Do | Don’t |
| --- | --- | --- |
| Workspace selection | Carry the workspace and authorized record identity across lists, routes, commands and background work. | Apply a stale result after the operator switched workspace. |
| Permission | Check current actor, role and object scope at the actual boundary. | Rely on hidden actions or a tenant name shown in the interface as the check. |
| Invitation | Bind scope, role, intended recipient policy and expiry to the invitation. | Treat possession of an expired or revoked reference as valid membership. |
| Approval | Preserve the exact version, target and authority reviewed. | Apply an old approval after its material inputs changed. |
| Bulk action | Show scope and return identifiable results for accepted, rejected or unresolved records. | Infer that all selected items succeeded from one successful request. |
| History | Record the actor, target and material outcome where history is required. | Expose secrets or full private payloads merely to populate audit entries. |
