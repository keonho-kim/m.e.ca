# Security and cryptographic decisions

Start from who can obtain database bytes, application execution, key-service access, snapshots and administrative privileges. A stronger primitive cannot repair an undefined trust boundary.

The [encrypted-document study](studies/encrypted-documents.md) connects these choices to pipeline ownership, manifest publication, DEK scope, KMS demand, buffer accounting and restore. Its numbers are assumptions, not defaults or measured guarantees.

| Need | Read |
| --- | --- |
| Field/object encryption, nonce safety, context binding and authenticated streams | [Encryption design](security/encryption.md) |
| KMS/HSM envelope hierarchy, rotation, recovery and deletion | [Key lifecycle](security/key-lifecycle.md) |
| Encrypted equality search, HPKE, post-quantum readiness, enclaves and encrypted computation | [Advanced privacy boundaries](security/advanced-privacy.md) |

Use the [identity reference](api/identity-and-tenancy.md) for tokens, workload/caller identity and tenant isolation. Keep authentication, authorization, transport security, storage encryption and audit separate: each has a different failure boundary.

## Match sophistication to a threat

| Threat or requirement | Candidate | Residual exposure |
| --- | --- | --- |
| Stolen disks/backups | Storage encryption and independent key custody | Authorized database/application reads |
| Database dump without application credentials | Field AEAD + envelope keys | Lengths, indexes and authorized decryption paths |
| Key-management operational failure | Versioned lifecycle, independent recovery and restore tests | Recovery access itself needs protection |
| Service should not see plaintext | Client-side encryption/recipient protocol | Search, recovery and collaboration change substantially |
| Untrusted host operator | Attested enclave plus constrained key release | Application bugs, side channels and release policy |
| Long confidentiality horizon | Crypto inventory and supported post-quantum protocol migration | Deployment interoperability, authentication and endpoint compromise |
| Computation must occur on ciphertext | Specialized homomorphic/MPC design with experts | Large cost, restricted computation and protocol assumptions |

Do not convert this table into default dependencies. Check algorithm/library support, compliance boundaries, errata and deployment constraints against the [security sources](sources/security.md). Random secrets need a cryptographic RNG; application timers, identifiers and hand-written ciphers do not substitute for one.
