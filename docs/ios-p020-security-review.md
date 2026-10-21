# P020 security review

## Scope and confirmed correction

Reviewed code source: `2ad963066aa918e86158e31c0b60d65bacb2218e`.
Initial exact hosted qualification source: `5819a82b8d628b66411335643c767e1e147de93d`.
Latest code source: `e7d3d36041d4bb49eeeb15ee4dcae2fe4d7d1391`; exact-source
P017, migration, native iOS, Procursus, and installer workflows passed. The CPU
helper/test adds no dependencies or auth path changes.

The post-upstream review found that `GET /api/beszel/universal-token` still
accepted `token`, `enable`, and `permanent` in its query string. This put a
universal token in a URL and allowed state changes through GET. P020 removes
GET mutation parsing and rejects any of those query keys with HTTP 400. GET
without these keys remains a read operation; the supported token-setting path
is POST with a JSON body. Tests assert independent rejection for all three
query keys, no token echo, no permanent record side effect, and inactive state
after rejection. Existing POST tests continue to cover role authorization,
expiry/persistence, disable, and Agent authentication.

This change prevents the application from accepting a universal-token value
through its GET URL contract. It cannot erase a URL that a caller has already
sent from access logs maintained by a reverse proxy or client. The frontend
uses POST for the supported flow; operators should treat any manually supplied
legacy token query URL as exposed and rotate that token.

## Go and dependency findings on the exact P020 source

P017 Release Candidate Regression run
[39605920035](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39605920035)
checked out exact source `5819a82b8d628b66411335643c767e1e147de93d` and passed:

- `GO-2026-6608` is absent from the Go 1.27.2 symbol scan.
- The scan records one module-level `GO-2026-5932` finding for
  `golang.org/x/crypto`; it remains open and is not claimed fixed. No OpenPGP
  import or symbol trace was reported in the reviewed source.
- Three raw `GO-2026-6809` traces remain in the scan, with Go's fixed release
  listed as 1.27.3. The official Go CL 851047 backport is present in the
  isolated Go 1.27.2 toolchain (`patch_sha256`
  `111bf066e27cb60df43881cbfa9b1d22b7682b720628b4b691a960a853b48150`), and
  `go test crypto/x509 -count=1` passed. The scanner retains the raw traces and
  records the backport disposition; the advisory is not hidden.
- Bun and npm reports retain `GHSA-vfj7-8cjw-p6xm` as OPEN. Both report zero
  critical findings. The advisory is build-only through the current frontend
  toolchain; this is a disposition, not remediation.

GitHub artifact metadata matches the exact-source evidence digests:

- Go vulnerability artifact `12347194497`, SHA-256
  `078d96ab1ffe6e7c20784f1bed1d4bab29f0a1af3bcc4a2a49c02982d46c23a0`.
- Bun/npm artifact `12347948106`, SHA-256
  `c73ce60b0c0d9516b4f28feba03135d32eea830f460586b67e79e9e151399f94`.
- P017 source/frontend manifest `12347404841`, SHA-256
  `2f0da23050a724149ff07e20c19fcd59dc8e3f576ed960cf0f8f1ae8fe05ef3f`.

The manifest reports source `5819a82`, 97 frontend assets, and states that no
device test, release publication, or tag creation occurred. P017 full
regression, P013 migration/restore and native iOS workflows all passed on this
same exact source.

## Native security and provenance

Native iOS run
[39605920505](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39605920505)
passed the Go 1.27.2 A7 runtime source patch and x509 backport, Agent and Hub
builds, ARM64 Mach-O inspection, iOS 12.0 minimum deployment check, VCS
provenance, checksums, embedded frontend verification and private MobileWiFi
linkage check. Both binaries report the exact source revision and
`vcs.modified=false`. The release job was skipped for this pull request.

The genuine P013 fixture on exact source `5819a82` used a synthetic database
initialized by the v0.19 Hub. It preserved accounts, role and SSH identity,
WAL-backed history, and verified full restore and interruption recovery. Its
12-file evidence artifact contains a synthetic validation manifest and
sanitized logs only; it does not contain the database archive or production data.
P013 reran on `e7d3d36` and passed the same upgrade, restart, restore, and
interruption-recovery checks. Its evidence artifact is `12349727839`, SHA-256
`75de188c4487c01e86508d512112b6ff526e6e09d5108e33c47124d3e3c236a1`.

## Latest exact-source disposition: `e7d3d36041d4bb49eeeb15ee4dcae2fe4d7d1391`

P017 regression run
[39609195215](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39609195215)
passed. Its source scan reports `target_advisory_in_findings=false` for
`GO-2026-6608`, three raw `GO-2026-6809` traces, and one module-level
`GO-2026-5932` finding. The official Go CL 851047 x509 backport is present;
the x509 test passes and the raw scanner traces remain recorded. `GO-2026-5932`
is not claimed fixed. Bun/npm report zero critical findings and retain
`GHSA-vfj7-8cjw-p6xm` as OPEN. The exact-source artifact digests are:

- Go vulnerability evidence artifact `12349927098`, SHA-256
  `aa994c110362e50ce3944cfc974ad95ddfed5f5ccca264cec4ddc1e876b602b1`.
- Bun/npm evidence artifact `12349258335`, SHA-256
  `6707d8261adb74e189435e8c73ba92bdb75c8b289f1568f56c0a6b70554c1c1f`.
- P017 manifest artifact `12348869369`, SHA-256
  `8f3e0678dc84809a076e416921b3c6aee54985f153bd5c3a707cf36ebe7fb66a`.

The manifest identifies source `e7d3d36`, 97 frontend assets, and records
`device_tested=false`, `release_published=false`, and `tag_created=false`.
Native iOS run
[39609195413](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39609195413)
also passed. Both binaries are Go 1.27.2 iOS/ARM64 device Mach-O files with
`GOARM64=v8.0`, `CGO_ENABLED=1`, minimum iOS 12.0, exact VCS revision
`e7d3d36041d4bb49eeeb15ee4dcae2fe4d7d1391`, and `vcs.modified=false`. SHA-256:

- Agent: `7956034cf1d0b0d1755cf082ccd021d3c0c40118a82fda723c4ee22c232ba1a5`.
- Hub: `727356fdc73a4d3985924dd394ba6f84f5cf7b9010e8941c4b7aabb689196bb5`.

The Hub embeds 97 verified assets; no private MobileWiFi linkage was present.
Payload artifact `12349709491` has SHA-256
`5b19ded34bbbfc94e9b3b68478541f93aa83a20e2310aa973fce920681df1492`;
native validation artifact `12349404872` has SHA-256
`2c7f46ab1c405cae969e9fd88539c3a0437bbb37323202fde44353a9477cf896`;
disabled Wi-Fi diagnostic artifact `12349884248` has SHA-256
`9f4041b702977adca9e75f630634354eefea65ce3a089baf06502bf489582d00`.
GitHub metadata matched these digests to their artifacts and run. The native
release job was skipped. The installer lifecycle workflow dispatch
`39609283025` also passed on this source.

The local attempt to retrieve the native validation archive was denied by its
GitHub signed blob URL with HTTP 403. The workflow logs and GitHub artifact
metadata were available; the CI validation job itself generated and uploaded
the evidence and verified the native binaries. No archive contents were
extracted into the local workspace.

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
