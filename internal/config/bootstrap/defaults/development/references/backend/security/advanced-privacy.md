# Advanced privacy and trust boundaries

Use only when an explicit threat or product requirement earns the complexity. These mechanisms have different goals; they are not interchangeable encryption-strength levels.

## Search without exposing a plaintext column

For exact equality, a keyed blind index can narrow candidates while field AEAD protects stored values. Separate index/encryption keys and bind index purpose, normalized representation and tenant scope. Rechecking decrypted candidates handles truncation collisions; a truncated index alone cannot establish exact uniqueness. [CipherSweet's blind-index design](https://ciphersweet.paragonie.com/internals/blind-index) is a concrete implementation model.

Equality/frequency, access patterns and permitted query behavior can still leak information. Low-entropy fields and many correlated indexes increase exposure. Do not replace this with an unkeyed plaintext hash, or promise arbitrary encrypted range/full-text search at normal index cost. Tink's [deterministic AEAD](https://developers.google.com/tink/deterministic-aead) explicitly exposes repeated-message equality. Order-preserving/revealing schemes need a leakage analysis and specialist review, not a default database option.

## Recipient and end-to-end encryption

Use a maintained recipient protocol where the service should not hold plaintext. [HPKE](https://datatracker.ietf.org/doc/html/rfc9180) composes a KEM, KDF and AEAD; it is an Informational CFRG RFC, and base mode is not sender authentication by itself. Identity binding, replay, membership changes, key distribution and recovery remain protocol decisions.

End-to-end encryption changes server-side search, previews, moderation, sharing and key recovery. Include client CPU/network and ciphertext/index overhead in the design. A server with a recovery key may still have a decryption path; describe the real custody model.

## Post-quantum readiness

Inventory public-key uses and confidentiality lifetimes, including transport, signatures, wrapped archives and third-party dependencies. NIST [ML-KEM](https://csrc.nist.gov/pubs/fips/203/final) establishes shared secrets; ML-DSA and SLH-DSA are signature standards described in NIST's [finalization announcement](https://www.nist.gov/news-events/news/2024/08/nist-releases-first-3-finalized-post-quantum-encryption-standards). They do not replace bulk AEAD with a new database cipher.

Use supported protocol/library migrations and inspect errata; do not invent a classical/post-quantum combiner. Budget larger messages, handshake CPU, packet fragmentation, key/certificate formats and ecosystem support. Crypto agility needs explicit allowed suites and retirement policy, without a downgrade fallback to an insecure suite.

## Confidential computation and privacy-preserving analysis

Attested enclaves can constrain key release to measured code; [Nitro Enclaves/KMS](https://docs.aws.amazon.com/enclaves/latest/user/kms.html) is one deployed mechanism. Evaluate attestation freshness, approved image updates, recovery, egress, side channels and the enclave's own code. An enclave does not make a vulnerable application safe.

Homomorphic computation can protect data from the evaluator but restricts computations and adds substantial CPU, memory and ciphertext costs. [Microsoft SEAL](https://github.com/microsoft/SEAL) provides specialized primitives; using it is a protocol/project decision, not a transparent SQL optimization. MPC or threshold protocols additionally require a precise adversary/collusion model and independently operated parties; use specialist implementations and a bounded pilot.

Differential privacy addresses information released by statistical queries rather than storage confidentiality. Bound contributions, account for composed privacy loss and test utility; repeated queries need a privacy-budget owner. [NIST SP 800-226](https://csrc.nist.gov/pubs/sp/800/226/final) explains evaluation hazards. Do not label noise, aggregation or anonymized identifiers alone as a privacy guarantee.

For any adoption, compare against a simpler trusted-service design, document residual leakage/trust and establish measurable latency, memory, storage and recovery acceptance conditions.
