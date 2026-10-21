# P021 security disposition

## Scope

This review covers the unified P001–P020 source at P020 integration merge
`42556e62c23f3595a276d2eb4a55125a96f5d31f`, pinned upstream
`b5357343550c8e2493d0e65472a06c1007396384`, and the P021 hosted Go and
Bun/npm scans that remain pending at the initial checkpoint. No advisory is
marked fixed without exact-source scan and build evidence.

## Findings carried forward from exact P020 source

P020 P017 run `39611915082` passed on final documentation head
`c716d3c9ce8de170cade25b3921bccef8c5b97a7`. The P020 code-source run
`39609195215` passed the same source audit on code SHA
`e7d3d36041d4bb49eeeb15ee4dcae2fe4d7d1391`.

| Advisory | Disposition |
|---|---|
| `GO-2026-6608` | Absent from the Go 1.27.2 symbol scan; exact-source P017 and native Go evidence passed. |
| `GO-2026-6809` | Three raw scanner traces remain visible. Official Go CL 851047 x509 backport is applied to the pinned Go 1.27.2 source and `crypto/x509` tests pass. The raw traces are retained, not suppressed. |
| `GO-2026-5932` | One module-level `golang.org/x/crypto` finding remains open. It has no OpenPGP call trace in the reviewed source; it is not claimed fixed. |
| `GHSA-vfj7-8cjw-p6xm` | Remains OPEN in Bun/npm audit evidence. The current disposition is a build-toolchain mitigation, not remediation. Zero critical JavaScript findings were reported. |

P020 exact-source evidence is documented in `docs/ios-p020-security-review.md`
and `docs/ios-p020-validation.json`. P021's own full Go and JavaScript scans
must pass before merge and must preserve any new raw finding.

## Application security review

The universal-token endpoint rejects `token`, `enable`, and `permanent` query
parameters on GET, returns a generic error without echoing a token, and keeps
mutation on POST JSON. Existing tests cover OAuth role preservation, record
ownership, universal-token authorization, token read-only rules, and alert
access. P020's endpoint correction and focused race tests passed; P021's full
source tests remain required.

Hosted evidence uses synthetic database records and source hashes only. No
production secrets, production data, device credentials, or private signing
material are part of the review artifacts.

## Pending exact P021 source

P021 P017 vulnerability scan, Bun/npm audit, native Go x509 backport test, and
final artifact/source metadata checks are PENDING. Until they pass on the final
P021 PR head, this document does not claim final P021 qualification.

P018 physical Gate F remains PENDING. No physical device, production branch,
release tag, GitHub release, or deployment is used by this cloud review.
