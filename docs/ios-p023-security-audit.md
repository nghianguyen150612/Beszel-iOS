# P023 security audit baseline

**Phase:** baseline established; P023 scans and remediation have not run yet.
**Source commit:** `fd334076f53d8354a4924a4c734f29cf65a43879`.
**Source tree:** `723e7261903f40e1d714f385f7bc3885f4691c2c`.
**Base:** `ios/upstream-v021-integration` at the verified P022 merge.

The source tree exactly matches the final P022 qualified head
`ef7d780cd1c548bb97cd1891810f945323983193`. P022 PR #26 reports Gate G PASS
and exact-head hosted migration, regression/security, native iOS/Procursus and
installer checks. P023 treats that as inherited evidence only; it will rerun
required checks against its own final source SHA.

## Starting exposure

| Finding | P022 evidence at the inherited tree | P023 baseline disposition |
|---|---|---|
| `GO-2026-6608` | Absent from the P022 Go 1.27.2 source scan. | Re-scan P023 with Go 1.27.3; retain absence only if reproduced. |
| `GO-2026-6809` | Three raw scanner occurrences remained. The official Go CL 851047 x509 backport was present and `go test crypto/x509 -count=1` passed. The advisory was not called absent. | Go 1.27.2 is affected; reconcile the backport against Go 1.27.3 source and test the native fix. |
| `GO-2026-5932` | `golang.org/x/crypto` module-level finding for unmaintained OpenPGP; no OpenPGP symbol trace was reported and no fixed module version was available. | Module is present (`v0.58.0`). P023 must enumerate imports and entrypoint reachability; module presence alone does not establish runtime exposure. |
| `GHSA-vfj7-8cjw-p6xm` | `braces@3.0.3` (affected through 3.0.3) was present in both Bun and npm locks through build tooling including `chokidar`/`micromatch` and Lingui. The Hub embeds generated assets. P022 did not qualify the proposed Lingui major upgrade. | Build-only graph, exact paths and build input trust still require P023 review. Keep the advisory open unless a real fixed release or qualified dependency change removes it. |

The P022 final PR record lists the Go scan archive as artifact `12392044916`
(SHA-256 `7c8d3d8068f5936e70f2b4ef58df2fed551b03653a755956e0d3b2a16cf19bf1`)
and JavaScript scan archive as artifact `12392708859` (SHA-256
`542cd22bc8a3c70d48eb2435c8c9f0a0c1b1e947019b7de7b635f79ce1bf1cb7`). P022
reported zero critical JavaScript findings. See [P022 release security](ios-p022-release-security.md),
[P022 machine disposition](ios-p022-security-disposition.json), and
[PR #26](https://github.com/nghianguyen150612/Beszel-iOS/pull/26) for the
inherited details. These artifacts do not substitute for P023's exact-head scan.

## Current dependency and pipeline facts

- `go.mod` declares Go 1.27.2 and `golang.org/x/crypto v0.58.0`.
- `internal/site/bun.lock` and `internal/site/package-lock.json` both resolve
  `braces@3.0.3`. The npm lock resolves Lingui CLI 5.9.5 while Bun's lock
  resolves 5.4.1; P023 will preserve and qualify both lockfile graphs.
- The iOS workflow applies a source-guarded Go 1.27.2 A7 runtime patch and a
  separately source-guarded x509 backport. The release metadata validators pin
  Go 1.27.2. P023 must update these contracts together.
- The release workflow is tag-push gated. P022 records PR publishing jobs as
  skipped and artifact provenance as same-run and source-SHA bound; P023 will
  check those properties again on its final PR head.

## Repository baseline and limits

GitHub refs were verified before P023 work: integration
`fd334076f53d8354a4924a4c734f29cf65a43879`, production `iOS`
`0c3e4cd22ac24264834c401ad412e44aeda4e4f3`, fork `main`
`e984ee12081f4e112601c3f718c9d81c112fe718`, and pinned upstream
`b5357343550c8e2493d0e65472a06c1007396384`. PR #26 is merged. The upstream
repository's current `main` is `b97abf7662a7c7a2dd7542ea2062405abd22d4dc`,
five commits ahead of the pinned snapshot. That drift is recorded and excluded
from P023.

At this baseline, P023 has not run new scanners, tests, migrations, installer
checks or native CI. No production branch, fork `main`, device, tag or release
has been changed. Gate H is pending. P018 Gate F remains pending, and this
baseline does not authorize a public release.
