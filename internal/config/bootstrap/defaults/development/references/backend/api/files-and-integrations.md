# Large objects and external integrations

Use when uploads, imports, exports or upstream calls can dominate memory, storage and execution time. Separate a small transactional manifest from large transferred bytes when their access/lifetime differ.

## Bound transfer and expansion

Limit encoded bytes, decompressed bytes, item count, nesting, dimensions and execution time at the appropriate parser/decoder boundary. A small compressed upload can expand beyond the service's capacity. Acquire weighted resources before allocating large working sets and release them on cancellation/error.

Stream or process bounded chunks when the format permits it; a streaming transport that first buffers the whole document has not reduced peak memory. Keep parsing, integrity checks, transformations and serialization from retaining duplicate complete bodies. Use private spool files only with byte quotas, cleanup ownership and a defined crash policy.

Uploaded content needs allowlisted formats, inspected type, generated safe names and controlled storage/serving. Parsing/previewing untrusted files may require a separate process or constrained worker when the decoder cannot enforce the needed boundary. See [OWASP upload guidance](https://cheatsheetseries.owasp.org/cheatsheets/File_Upload_Cheat_Sheet.html).

## Publish objects deliberately

Upload to a private immutable object identity, verify integrity/required processing, then commit the manifest/visibility change. Reconcile failure between object storage and the database: clean confirmed orphans after a grace/ownership check, not objects with uncertain active publication. A visible object should not point to unfinished bytes.

Presigned URLs are bearer capabilities with bounded scope/lifetime. They can be used more than once while valid and can overwrite a chosen key if permitted; use server-owned immutable keys or explicit conditional/version semantics. Avoid logging the URL. See [S3 presigned URL semantics](https://docs.aws.amazon.com/AmazonS3/latest/userguide/using-presigned-url.html).

Exports require current authorization, snapshot/live semantics, resumability, result expiry and bounded temporary storage. Pagination improves resource use only if the export's consistency model tolerates changes between pages; otherwise choose a managed snapshot and budget its age/maintenance impact.

## Keep integrations inside their boundary

Reuse correctly configured clients/pools with connection, header/body, read-idle and total-operation budgets. Propagate cancellation; close response bodies and cap error bodies before logging/decoding. Bound fan-out and account for SDK retries and reconnect bursts. Circuit/fallback behavior must preserve the product's consistency and authorization semantics.

For user-controlled URLs/destinations, enforce policy on the actual resolved connection and redirects, including IPv4/IPv6/private/link-local/metadata targets and DNS changes. A string prefix or pre-resolution check alone does not contain egress. Use network controls when required; see [SSRF prevention](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html).

Validate malformed/expanding files, disk full, cancellation mid-transfer, lost final acknowledgement, failed manifest commit, orphan cleanup race and hostile redirects. Measure peak retained bytes, temporary bytes, open descriptors, upstream connection reuse and useful throughput during concurrent large work.
