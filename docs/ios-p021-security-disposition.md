# P021 security disposition

## Scope

This review covers candidate source `037ffa8457cbd33b7444a0bc9035232c0b7502c6`, the pinned upstream `b5357343550c8e2493d0e65472a06c1007396384`, and exact-source P017/native CI. P017 run `39614985449` and native run `39614985052` passed. Security findings below remain open where stated; no advisory is called fixed without exact-source scan and build/test evidence.

## Go and JavaScript findings

| Advisory | P021 disposition and evidence |
|---|---|
| `GO-2026-6608` | Absent from the Go 1.27.2 scan on source `037ffa8`; P017 scan passed. |
| `GO-2026-6809` | Three raw scanner traces remain visible. Official Go CL 851047 x509 backport is applied to pinned Go 1.27.2; `crypto/x509` tests passed in P017 and native CI. Raw traces are preserved, not suppressed. Patch SHA-256: `111bf066e27cb60df43881cbfa9b1d22b7682b720628b4b691a960a853b48150`. |
| `GO-2026-5932` | One module-level `golang.org/x/crypto` finding remains open. The reviewed source has no OpenPGP call trace; it is not claimed fixed. |
| `GHSA-vfj7-8cjw-p6xm` | Remains OPEN. P017 Bun/npm audit reported zero critical JavaScript findings. This is a build-toolchain mitigation, not remediation. |

P017 artifacts: Go vulnerability evidence `12351029995` (SHA-256 `0a52e6d75699f87b2aacafc6239686bd005235565825a50b288342ff8c890cfe`), JavaScript audit evidence `12351723962` (SHA-256 `bdcc7a16f9aa4bfbc212fa92d9855f6af92bf5a31b875ffffea81d873d63c3ce`), and regression manifest `12352223742` (SHA-256 `01586c39d693dda3ff4c006f0c485400d014edc6eacd191fba60ec9f336582f5`). All belong to P017 run `39614985449`, whose candidate head is `037ffa8`.

The native run applied and tested the same CL 851047 x509 backport against Go 1.27.2. Its iOS payload artifact and validation evidence are listed in `ios-p021-validation.json`.

## Application security review

The universal-token endpoint rejects `token`, `enable`, and `permanent` query parameters on GET, returns a generic error without echoing a token, and keeps mutation on POST JSON. P017's full Go suite passed on the candidate source. Coverage includes OAuth role preservation, protected ownership relationships, universal-token authorization, read-only GET behavior, and alert access. P013 used synthetic disposable database records for migration and restore; no production records or secrets were used.

Hosted evidence contains source hashes and synthetic test data only. No production secrets, production data, device credentials, or private signing material are included.

## Merge disposition

P021's required Go, JavaScript, native, migration, Procursus, and installer checks passed on source `037ffa8457cbd33b7444a0bc9035232c0b7502c6`. The open advisories above are retained and documented; zero critical JavaScript findings were reported, and no critical unresolved regression was identified. The final checkpoint includes only P021 documentation and CI path-filter changes, so its exact PR-head checks must pass before merge. See the exact final-head check suite and closeout comment on PR #25.

P018 physical Gate F remains PENDING. No physical device, production branch, release tag, GitHub release, or deployment is used by this cloud review.
