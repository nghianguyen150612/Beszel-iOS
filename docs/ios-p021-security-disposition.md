# P021 security disposition

## Scope

This review covers candidate source `b20ed2b636d95bca26c87161de06d8e1e7f03862`, pinned upstream `b5357343550c8e2493d0e65472a06c1007396384`, and exact-source P017/native CI. P017 run `39620427058` and native run `39620426859` passed. No advisory is called fixed without exact-source scan and build/test evidence.

## Go and JavaScript findings

| Advisory | P021 disposition and evidence |
|---|---|
| `GO-2026-6608` | Absent from Go 1.27.2 findings (`target_advisory_in_findings=false`) on `b20ed2b`; P017 scan passed. |
| `GO-2026-6809` | Three raw scanner occurrences remain. Official Go CL 851047 x509 backport is applied to pinned Go 1.27.2; `crypto/x509` tests passed in P017 and native CI. Raw traces are retained. Patch SHA-256: `111bf066e27cb60df43881cbfa9b1d22b7682b720628b4b691a960a853b48150`. |
| `GO-2026-5932` | One module-level `golang.org/x/crypto` finding remains open. The reviewed source has no OpenPGP call trace; it is not claimed fixed. |
| `GHSA-vfj7-8cjw-p6xm` | Remains OPEN. P017 Bun/npm audit reported zero critical JavaScript findings. The current measure is a build-toolchain mitigation, not remediation. |

P017 run `39620427058` artifacts: Go vulnerability evidence `12354258581` (SHA-256 `615e309fb58a59f3b570f857125693f4eb059ea259e9ba7fbe05a2ba51216402`), JavaScript audit evidence `12354567142` (SHA-256 `073556df0c75b6d54f76baf572b94c987c6b9884c753ac9c6db314220b16b686`), and regression manifest `12355115796` (SHA-256 `9cb52a17b3fc1d5fdc053309f6ce7deb079785be2e1446982fa8f614952d41d4`). The P017 manifest is tied to candidate `b20ed2b` and records 97 frontend assets.

The native run tested the same CL 851047 x509 backport with Go 1.27.2. Its payload and validation artifact digests appear in `ios-p021-validation.json`.

## Application security review

The universal-token endpoint rejects `token`, `enable`, and `permanent` query parameters on GET, returns a generic error without echoing a token, and keeps mutation on POST JSON. P017's full Go suite passed on `b20ed2b`. Coverage includes OAuth role preservation, protected ownership relationships, universal-token authorization, read-only GET behavior, and alert access. P013 used synthetic disposable database records; no production records or secrets were used.

Hosted evidence contains source hashes and synthetic test data only. No production secrets, production data, device credentials, or private signing material are included.

## Merge disposition

P017, P013, native iOS/Procursus, and installer checks passed on candidate `b20ed2b`. The open advisories above remain documented; zero critical JavaScript findings were reported, and no critical unresolved regression was identified. The final P021 evidence commit triggers all four workflows again; each exact PR-head run must pass before merge. See the PR check suite and final closeout comment on PR #25.

P018 physical Gate F remains PENDING. No physical device, production branch, release tag, GitHub release, or deployment is used by this cloud review.
