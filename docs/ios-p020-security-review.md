# P020 security review

## Scope and confirmed correction

Reviewed source: `2ad963066aa918e86158e31c0b60d65bacb2218e` on branch
`ios/p020-post-upstream-qualification`.

The post-upstream review found that `GET /api/beszel/universal-token` still
accepted `token`, `enable`, and `permanent` in its query string. This put a
universal token in a URL and allowed state changes through GET. P020 removes
GET mutation parsing and rejects any of those query keys with HTTP 400. GET
without these keys remains a read operation; the supported token-setting path
is POST with a JSON body. Tests assert independent rejection for all three
query keys, no token echo, no permanent record side effect, and inactive state
after rejection. Existing POST tests continue to cover role authorization,
expiry/persistence, disable, and agent authentication.

This change prevents the application from accepting a universal-token value
through its GET URL contract. It cannot erase a URL that a caller has already
sent from access logs maintained by a reverse proxy or client. The frontend
uses POST for the supported flow; operators should treat any manually supplied
legacy token query URL as exposed and rotate that token.

## Go and dependency findings on the P020 source

P017 Release Candidate Regression run
[39604710829](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39604710829)
checked out exact source `2ad963066aa918e86158e31c0b60d65bacb2218e` and
completed its scan steps before the full regression job finished:

- `GO-2026-6608` is absent from the Go 1.27.2 symbol scan.
- The scan records one module-level `GO-2026-5932` finding for
  `golang.org/x/crypto`; no OpenPGP import or symbol trace was reported. It
  remains open and is not claimed fixed.
- Three `GO-2026-6809` traces are reported, with Go's fixed release listed as
  1.27.3. The official Go CL 851047 backport is present in the isolated
  Go 1.27.2 toolchain (`patch_sha256`
  `111bf066e27cb60df43881cbfa9b1d22b7682b720628b4b691a960a853b48150`), and
  `go test crypto/x509 -count=1` passed. The scanner retains the raw traces and
  records the backport disposition; the finding is not hidden.
- Bun and npm reports retain `GHSA-vfj7-8cjw-p6xm` as OPEN. Both report zero
  critical findings. The advisory is build-only through the current frontend
  toolchain; this is a disposition, not remediation.

The hosted evidence artifact digests reported by the run are:

- Go vulnerability evidence, artifact 12348241118:
  `70b4e6353dcd7623cd9e29e7f4b62351cd8b0436bfad19f3590f12ff89d464f6`.
- Bun/npm dependency evidence, artifact 12348022491:
  `9609492918d781f0e3be51c39d3e14379ca5cd4b68ae7c9fc739ae0cf8527665`.

The run also reported zero critical JavaScript findings and uploaded a
candidate source manifest. P017 full regression, P013 migration and native
iOS checks remain separate required gates and were still running when this
review checkpoint was written.

## Authorization, logs, and artifacts

The new rejection path returns a generic error without echoing the supplied
token. The route implementation has no request-URL logging. This source review
is not a claim that external proxies suppress URL logging; the endpoint-level
test is the application boundary. Existing deterministic tests cover record
ownership reassignment protections, OAuth role preservation, universal-token
authorization, and token collection read-only rules.

Hosted reports contain source hashes and vulnerability metadata, not
production secrets or production data. Native binaries are review artifacts;
this task does not install or execute them on a physical device. No release,
tag, deployment, production branch update, or fork `main` update is in scope.
P018 physical Gate F remains pending.
