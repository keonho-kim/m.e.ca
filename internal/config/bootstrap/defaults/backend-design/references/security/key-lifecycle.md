# Key hierarchy, rotation and recovery

Use when encryption must remain available through years of writes, restarts, backups, credential changes and incidents. Key management is a durable state machine with an external trust boundary; a periodic random-key generator is not sufficient.

## Separate data keys and wrapping authority

Envelope encryption uses data-encryption keys (DEKs) for payloads and a key-encryption authority (KEK/KMS/HSM) to protect them. Choose DEK scope from blast radius, bytes/messages, erasure needs and KMS cost: per-object, tenant or bounded epoch each has different metadata/cache overhead. AWS's [data-key model](https://docs.aws.amazon.com/kms/latest/developerguide/data-keys.html) illustrates plaintext and wrapped copies and makes clear that the application owns data-key handling.

Persist wrapped DEK, immutable key identity/version and authenticated context with a recoverable manifest. Keep wrapping authority independent from a database dump; restrict who may decrypt and administer/delete keys. A KMS/HSM reduces exposure of the wrapping secret, while the application may still receive DEKs/plaintext. Audit use without recording those secrets.

HKDF can separate purpose-specific keys from suitable high-entropy material using defined context; it is not a password-hardening function or automatic entropy source. See [HKDF](https://www.rfc-editor.org/rfc/rfc5869.html). Password verification needs an appropriate memory-hard implementation, tuned under bounded concurrent admission; [Argon2](https://www.rfc-editor.org/rfc/rfc9106.html) is an Informational CFRG specification, not a universal parameter mandate. A high-entropy random API token may need only a protected one-way verifier, whereas an upstream credential the service must present again needs recoverable encryption; do not impose password-hashing cost on every secret by habit.

## Make publication and rotation recoverable

Use explicit states such as prepared → current-for-write → read-only → retirement-eligible → destroyed, adapting names to the storage system. Store identity before publication can lose it; verify that the key is retrievable before making new ciphertext depend on it. Across key-service/database boundaries, use durable intent, idempotent identity and reconciliation rather than pretending one transaction covers both.

With several writers, serialize or compare-and-set the active epoch. A rotation worker needs bounded work, an ownership/fencing rule, crash recovery and jitter where many tenants would rotate together. New writes use the established current key; existing ciphertext remains readable until its actual dependencies are gone.

Distinguish three operations: rotate the wrapping authority's material, rewrap existing DEKs, and re-encrypt payloads with new DEKs. They have different security effects and I/O costs. AWS [KMS rotation](https://docs.aws.amazon.com/kms/latest/developerguide/rotate-keys.html) can retain old material for decryption; that does not mean application payloads were rewritten or old compromise exposure disappeared.

Choose rotation from cryptoperiod, exposure, algorithm limits and incident policy, not a magic interval. Limit key caches by age, uses, bytes and memory; a KMS outage or cache miss must have a deliberate fail-closed/availability outcome. Avoid a full-table rewrite on the request path; use resumable bounded work and observable backlog when re-encryption is required.

## Recovery precedes destruction

Inventory live data, old snapshots, replicas, archives, exports and retained wrapped keys before retirement. Test restoring old backups with the corresponding permitted key access. Crypto-erasure requires destroying every usable key copy/recovery path; deleting an active-key pointer is insufficient. Protect recovery access with separate custody/authorization and auditable use. Consult [NIST key-management guidance](https://csrc.nist.gov/pubs/sp/800/57/pt1/r5/final).

Verify failures before/after key creation, ciphertext commit and epoch switch; concurrent rotation, stale writers, missing permissions, KMS outage and interrupted rewrap. Unknown publication must not trigger blind deletion of a key that may protect committed data.
