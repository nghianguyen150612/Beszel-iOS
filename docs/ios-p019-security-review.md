# P019 security review

Status: **PASS on code source `bc6967dc3cffe2d46be5b427aa490e87227ab571`**.
Documentation changes after this source require the final PR-head checks to
pass before merge.

## Go standard library

The pinned Go version remains 1.27.2. The authoritative Go vulnerability entry
`GO-2026-6608` affects `net/textproto` and `mime/multipart` with a multipart MIME
header memory limit bypass; it is fixed in Go 1.27.2 and is absent from the
exact-source P019 scan.

The Go 1.27.2 scan also reports three reachable `GO-2026-6809` traces through
`crypto/x509.Certificate.Verify`, TLS chain handling, and
`internal/ghupdate.FetchLatestRelease`. The advisory, CVE-2026-111943, concerns
distrust of compromised intermediate certificates; Go lists 1.27.3 as the
fixed release. P019 backports the official Go CL 851047, commit
`7f86f957633ee31a1c17a1eb8452ca7a3d9bc141`, while retaining Go 1.27.2 and the
separate A7 runtime patch.

The official patch is stored as
`patches/go/1.27.2/GO-2026-6809.patch.b64`. Its decoded SHA-256 is
`111bf066e27cb60df43881cbfa9b1d22b7682b720628b4b691a960a853b48150`. The
fail-closed patcher validates exact baseline source hashes, the complete
eight-entry denylist and platform hooks, then records patched source hashes.
Both P017 and native iOS workflows apply it only to an isolated GOROOT and run
`go test crypto/x509 -count=1`. Local Linux, hosted Ubuntu, and hosted macOS
standard-library tests passed. The native Agent and Hub builds use that patched
GOROOT; they report Go 1.27.2 and retain the A7 runtime validation.

Hosted P017 run
[39596633842](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39596633842)
used govulncheck v1.9.0 and the Go vulnerability database last modified
`2026-10-21T00:27:29Z`. The scan retains four occurrences: three
`GO-2026-6809` traces and one module-level `GO-2026-5932` finding. Its validated
summary records the upstream backport disposition for GO-2026-6809 only when
the exact patch manifest and passing x509 test are present; it does not remove
or hide the raw scanner findings. GO-2026-6608 is absent.

`GO-2026-5932` is a module-level finding for `golang.org/x/crypto` v0.58.0
without a symbol trace. The advisory concerns OpenPGP; no OpenPGP imports were
found in repository source. It remains recorded without claiming remediation.

## JavaScript build dependency

The exact-source Bun/npm audits report zero critical findings and keep
`GHSA-vfj7-8cjw-p6xm` **OPEN**, as required. The affected `braces` dependency is
build-only through Lingui. The scanner summary retains the advisory rather than
claiming it is fixed. On source `bc6967dc`, the audit evidence artifact digest
is `0ee40c987cd6b5ebce5c1bda53a26f81033af81495c8668fa55bc6d83cbf73cb`.

## Additional findings and boundaries

The Go source report, JS audits, and patch manifests contain no production
secrets or data. The hosted native artifacts are unsigned review artifacts;
they were not installed or run on a physical device. The native job's release
step was skipped. P018 physical-device Gate F remains pending. No release,
deployment, tag, or device action was part of this review.
