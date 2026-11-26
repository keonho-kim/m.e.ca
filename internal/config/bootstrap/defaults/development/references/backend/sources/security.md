# Security primary sources

Inspect publication status, errata, library versions and actual cryptographic-module validation before adopting a policy. These pointers support the [security map](../security.md); no algorithms, sample secrets or implementation code are copied into the bundle.

## Primitives and key custody

| Source | Status/scope and useful decision |
| --- | --- |
| [NIST SP 800-57 Part 1 Rev. 5](https://csrc.nist.gov/pubs/sp/800/57/pt1/r5/final) | Key lifecycle, cryptoperiod, protection and recovery guidance |
| [NIST SP 800-38D](https://csrc.nist.gov/pubs/sp/800/38/d/final) | GCM/GMAC; publication includes a revision planning note |
| [Tink AEAD](https://developers.google.com/tink/aead) | Authenticated encryption and visible associated-data binding |
| [Tink deterministic AEAD](https://developers.google.com/tink/deterministic-aead) | Equality leakage and deterministic authenticated encryption |
| [Tink streaming AEAD](https://developers.google.com/tink/streaming-aead) | Authenticated segmented streams and supported key types |
| [libsodium AEAD](https://doc.libsodium.org/secret-key_cryptography/aead) | Construction interoperability, nonce sizes and usage limits |
| [RFC 5869](https://www.rfc-editor.org/rfc/rfc5869.html) | Informational HKDF key derivation and context separation |
| [RFC 9106](https://www.rfc-editor.org/rfc/rfc9106.html) | Informational CFRG Argon2 specification and test-vector reference |
| [RFC 9180](https://datatracker.ietf.org/doc/html/rfc9180) | Informational CFRG HPKE; recipient modes and composition |
| [RFC 8446](https://www.rfc-editor.org/rfc/rfc8446.html) | Standards Track TLS 1.3; transport and early-data replay considerations |
| [AWS KMS concepts](https://docs.aws.amazon.com/kms/latest/developerguide/concepts.html) | Wrapping authority, key identity, administration and custody |
| [AWS KMS data keys](https://docs.aws.amazon.com/kms/latest/developerguide/data-keys.html) | Application-owned data-key use and wrapped-key storage |
| [AWS KMS rotation](https://docs.aws.amazon.com/kms/latest/developerguide/rotate-keys.html) | Key-material rotation versus application data re-encryption |

XChaCha20-Poly1305 is documented as widely implemented but not standardized in the referenced libsodium construction table. HPKE/HKDF/Argon2 RFC publication does not give them the same status as an Internet Standards Track protocol. Algorithm choice does not establish FIPS validation of a selected library/deployment.

## Specialized privacy/trust boundaries

| Source | Scope and useful decision |
| --- | --- |
| [CipherSweet blind indexes](https://ciphersweet.paragonie.com/internals/blind-index) | Keyed searchable-field candidate indexes and truncation |
| [NIST FIPS 203 / ML-KEM](https://csrc.nist.gov/pubs/fips/203/final) | Final KEM standard with a potential-errata note |
| [NIST PQ finalization announcement](https://www.nist.gov/news-events/news/2024/08/nist-releases-first-3-finalized-post-quantum-encryption-standards) | Accessible primary summary of ML-KEM, ML-DSA and SLH-DSA roles |
| [Nitro Enclaves/KMS attestation](https://docs.aws.amazon.com/enclaves/latest/user/kms.html) | Measured-code-based key-release boundary |
| [Microsoft SEAL](https://github.com/microsoft/SEAL) | Maintained homomorphic-computation primitives and limitations |
| [NIST SP 800-226](https://csrc.nist.gov/pubs/sp/800/226/final) | Final March 2025 guidance on differential-privacy evaluation hazards |

The individual FIPS 204/205 publication pages could not be retrieved here; their full algorithm specifications were not reviewed. Use the accessible NIST summary only for the role distinction and verify exact specifications before implementation. Specialized privacy techniques require a protocol/threat analysis beyond a library feature checklist.
