# Encryption design and execution cost

Use when protecting fields, objects or streams. Define the attacker and plaintext boundary first: disks/backups, database operators, application code, key-service principals and clients have different access. Storage encryption cannot hide data from an authorized application that decrypts it.

## Choose a complete construction

Prefer a maintained AEAD implementation such as a suitable Tink/libsodium/platform primitive. AES-GCM can benefit from hardware acceleration; ChaCha20-Poly1305/XChaCha20-Poly1305 have different hardware, nonce and interoperability properties. Benchmark representative message sizes on target CPUs and verify compliance/library support rather than inventing a universal winner. See [Tink AEAD](https://developers.google.com/tink/aead) and [libsodium constructions](https://doc.libsodium.org/secret-key_cryptography/aead).

Use a cryptographic RNG and the construction's nonce API correctly. Enforce message/byte/per-key limits from the selected implementation and security analysis. GCM nonce reuse under the same key is catastrophic; counters need unique allocation across processes, restarts, forks and snapshot restore, while random nonces need collision-budget accounting. Rotation does not repair past reuse. Follow [GCM guidance](https://csrc.nist.gov/pubs/sp/800/38/d/final) and current errata.

Bind ciphertext to canonical associated data such as tenant, object, field, format and intended purpose. Associated data is authenticated but visible. Reconstruct the expected binding from trusted context; accepting the attacker's copied header as the whole binding defeats row-swap protection. Reject authentication failure without returning partial unauthenticated data or silently recreating a value.

Design a versioned, bounded envelope with an allowlisted algorithm/key identifier and integrity-protected interpretation. Do not select arbitrary algorithms or key-service endpoints from untrusted ciphertext metadata. Preserve unsupported/corrupt user data and report a safe registered fault.

## Encrypt where it saves and protects

Encrypt immutable large content once and reference it from metadata when lifetime/scope permit. Avoid decrypting entire bodies for lists, counting or logs. Batch authorized metadata lookup and bounded decrypt work; do not cache plaintext without a lifetime, byte budget and explicit threat decision. Plaintext key caching is a separate, more sensitive budget.

Compress before encryption only when format, trust and confidentiality permit it. Ciphertext is normally incompressible, and combining attacker-controlled input with secrets before compression can reveal information through length. Consider length leakage and padding from the actual threat, including the extra bytes and transfer cost.

For large files, use an established authenticated streaming construction that binds segment identity/order and detects truncation according to its contract. Do not independently encrypt chunks with ad hoc repeated nonces. Tink's [streaming AEAD](https://developers.google.com/tink/streaming-aead) documents segmented authentication; replacing an entire older valid ciphertext still requires an external freshness/version authority.

## Verify the real boundaries

Test wrong tenant/object/field, modified header/body, invalid tag, truncated/reordered stream, unsupported version, missing key and restart/restore behavior. Benchmark ciphertext overhead, allocation/copy count, throughput, tail latency and key-service calls separately. Do not log plaintext, key material or full decrypted causes. Use [key lifecycle](key-lifecycle.md) for recoverability and [advanced privacy](advanced-privacy.md) when equality search or a different trust model is required.
