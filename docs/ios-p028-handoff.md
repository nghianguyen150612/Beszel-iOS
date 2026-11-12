# P028 dependency, migration and upstream compatibility

Audit date: 2026-11-11. This is implementation on `ios/p028-dependency-migration`,
stacked on verified P027 `c239ce0d8931405f728ef4763ad68bbbe187a9ac`.
Draft PR: https://github.com/nghianguyen150612/Beszel-iOS/pull/32,
target `ios/p027-upstream-port`. Production `iOS` remains
`0c3e4cd22ac24264834c401ad412e44aeda4e4f3`; P026 and P027 remain unmerged.

Official `henrygd/beszel` main is
`800d18bbb5fa2240b6dbc7e155f3af0c34c01ad5`, unchanged from the P027 snapshot.
Its application version is 0.21.1. The official branch inventory has no
separate stable 0.21 maintenance branch; this batch follows the verified main
changes for that release line. No divergent feature branches were merged.
The updated P027 ledger retains full upstream SHAs and individual dispositions.

## Implementation and compatibility

| Dependency | P027 | P028 |
| --- | --- | --- |
| PocketBase | 0.40.5 | 0.40.6 |
| CBOR | 2.9.6 | 2.10.0 |
| shoutrrr | 0.22.0 | 0.22.1 |
| x/net | 0.62.0 | 0.62.0, retained |
| Go | 1.27.3 | 1.27.4 |

All three dependency changes come from upstream
`5750cead8db77ae5aeebea98a0db29d976116728`. The 0.40.5/0.40.6 module source
comparison found no changes to core migration, authentication, collection,
server lifecycle or SQLite code. Changes include filesystem bucket escape
defenses, thumbnail pixel limits, upstream update delivery, JSVM dependencies
and the embedded PocketBase admin UI. Beszel's Hub continues to register its
own migrations, collection rules and command handling. No application schema,
SQLite implementation, collection field or session policy was changed by P028.
This source comparison is supporting evidence, not a substitute for migration tests.

CBOR's release adds performance improvements and opt-in functionality. No
encoder options or struct tags are changed here. A fixed telemetry fixture was
generated from verified P027 source using CBOR 2.9.6; the 2.10.0 regression
decodes it and reproduces the exact bytes. It includes Battery, CPU byte
arrays, memory/swap byte tuples, fans and Wi-Fi. Existing transport, absent-field
and JSON v1/v2 tests remain in place. Shoutrrr tests exercise its actual service
with the injected HTTP client, successful delivery, HTTP failure and deadline
propagation. Existing destination checks, DNS/redirect protections, role checks
and pre/post-initialization injection remain intact.

Portable ports include:

- `161f009b`: upstream `no_self_update`, adapted so iOS always refuses upstream
  binary replacement. iOS native payloads explicitly require this build tag;
  the artifact validator checks it. Upstream replacement bypasses the port's
  A7 runtime, signing, package ownership and checkpoint/rollback lifecycle.
  Other platforms retain their existing default update behavior.
- `b4195bdd`: `-k` accepts inline SSH public keys or a key-file path, with
  multi-key, spaces, invalid-content and missing-file regressions.
- `e0fc9886`: preserve P026's stricter Battery decoder in `byte_metrics.go`
  and add upstream JSON v1/v2 and API regressions. Numeric tuples, older
  base64 payloads, overflow/length checks and absent values remain supported.
- `55b61da0`: remove a moved monitor from its persisted old system and ignore
  stale results for deleted monitors. A regression distinguishes the saved
  source from a pending destination; prior ownership checks remain intact.
- `5aaa7aee`: shared fan filter configuration and its upstream tests.
- `f6e4ad4a`: HTML alert/auth email layout while retaining plain text and
  existing notification access checks; upstream escaping/template tests remain.
- `b3a2fe91`, `0df763ef`: swap axis ticks and missing SMART values. The swap
  adaptation retains P026 `memoryFieldBytes` and measured/absent semantics.

iOS returns `errors.ErrUnsupported` in `newGPUManagerForPlatform` before GPU
collector discovery. GPU utility retries/backoff, AMD sysfs retry/backoff and
Linux-only RAID/DNF/systemd/SELinux changes have no iOS execution path and were
not introduced into it. The ledger explicitly leaves non-iOS collector parity
unported. Remaining portable pagination, locale and other ledger entries are
still outstanding; this is not a claim that all upstream work is finished.

## Synthetic migration qualification

`tests/hub-migration/harness.py` builds genuine historical production v0.19,
the candidate and, for `--via-p027`, pinned P027. Each gets its real frozen-lockfile
production frontend. Only synthetic credentials and databases are used.

The direct and P027-intermediate CI matrix paths cover fresh bootstrap, original
v0.19 migration, an already-migrated PocketBase 0.40.5 database, second startup,
original migration ledger, records/relationships, collection schema, existing
session refresh, main and auxiliary SQLite integrity, API authorization/history,
rollup/retention, five injected failure/recovery checkpoints and compatible
restart. The intermediate path requires unchanged schema, record counts,
settings and migration history across the PocketBase patch upgrade. Database
archives and private keys stay disposable; only redacted manifests are uploaded.

An initial hosted attempt rejected P027's admin-only sharing rule because the
historical validator predated that hardening. The validator now requires the
exact P027 rule. It does not accept arbitrary rule changes. Earlier attempts
also caught a duplicate upstream Battery method and a native build-info parser
that did not recognize `-tags`; both were corrected without weakening the
original decoder or artifact checks.

## Earlier hosted migration evidence — source 43e2b92

Both matrix paths passed on exact source
`43e2b92f321faf4fa9c37f2a8723d5cee1536566`, run
[42953384735](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/42953384735).
The direct job is 130302699412; the P027-intermediate job is 130302699170.
The original 12 migration entries are preserved and the final ledger has 19,
with ordered application additions present exactly once. Both healthy starts
return HTTP 200; `data.db` and `auxiliary.db` integrity checks return `ok` and
foreign-key checks return no violations. The intermediate reports unchanged
schema/records/settings/ledger and existing-session refresh HTTP 200. Each path
passes all five injected failure/recovery checkpoints and complete v0.19 rollback.

Full redacted manifests are `ios-p028-migration-direct.json` and
`ios-p028-migration-via.json`. Both artifacts were independently downloaded and
verified against GitHub's archive digest:

| Path | Artifact | Archive SHA-256 |
| --- | --- | --- |
| Direct | 13825813979 | `6af61ca318d61c8cefd31fae16f820deab2126f65fa37090dd031332ce38c95d` |
| Via P027 | 13826511696 | `48af8e3bb2e60001f559e6dd28ee8902b303c273dd96750299b18944d4cc4fa8` |

PocketBase upgrade qualification is accepted for this synthetic host scope.
This does not qualify an owner database or a physical iOS migration.

## Historical validation checkpoint — source b8722c5d

Repository: /workspace/work/Beszel-iOS. Branch: ios/p028-dependency-migration.
Tested implementation source HEAD and pushed branch at this checkpoint:
b8722c5d94b42e9331e12acf399f9d00fd638b53. P028 base:
c239ce0d8931405f728ef4763ad68bbbe187a9ac. PR #32 remains open Draft and
targets ios/p027-upstream-port. Production iOS remains
0c3e4cd22ac24264834c401ad412e44aeda4e4f3; P027 remains unmerged.
The handoff/evidence update is documentation-only; validated source is still
b8722c5d94b42e9331e12acf399f9d00fd638b53.

P028 Go update commit: 1d1ba4e96918fed01321bf572d005b15122208dc.
Locale synchronization commit: b8722c5d94b42e9331e12acf399f9d00fd638b53.
The refreshed Go advisory database reported GO-2026-6859, fixed in Go 1.27.4.
The exact Go 1.27.4 symbol scan no longer reports GO-2026-6859, GO-2026-6809,
or GO-2026-6608. It retains only module-level GO-2026-5932 in
golang.org/x/crypto@v0.58.0, with no affected package import or reachable
symbol. The OpenPGP import guard passed six host/iOS profiles, but a release
owner decision remains required. This is not described as a dependency fix.

Local validation on the implementation source:

- bun install --frozen-lockfile: PASS.
- bun run sync: PASS; 34 locale catalogs synchronized and reviewed; committed.
- bun test: PASS, 65 tests / 320 assertions.
- bun run build: PASS; Vite reports its existing large WebAssembly chunk warning.
- scripts/audit/validate_frontend_assets.py --dist internal/site/dist --write-manifest /workspace/work/beszel-assets-p028.generated.sha256: PASS; 99 files, 3,431,143 bytes, 34 locales; manifest matches rebuilt output.
- env -u NETWORK PATH="/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.4.linux-amd64/bin:$PATH" GOTOOLCHAIN=local go test -count=1 -p 1 -tags='testing no_ui no_self_update' ./...: PASS.
- Same Go environment, go vet -p 1 -tags='testing no_ui no_self_update' ./...: PASS.
- Same Go environment, go mod verify: PASS, all modules verified.
- Same Go environment, go mod tidy -diff: PASS, no diff.
- python3 -m unittest discover -s tests/audit -p 'test_*.py': PASS, 137 tests.
- go test crypto/x509 -count=1 against official Go 1.27.4: PASS.
- govulncheck v1.9.0 Go 1.27.4 symbol scan: scan process exit 0; exact report has only module-level GO-2026-5932. Summary is docs/ios-p028-govulncheck-go1274.json.

Exact-source hosted qualification for b8722c5d94b42e9331e12acf399f9d00fd638b53:

- P013 Hub migration run 42980635598: direct job 130389963579 PASS and
  P027-intermediate job 130389963233 PASS. Direct artifact 13837835918 was
  independently downloaded and matched GitHub SHA-256
  f27b0eebc25f0ba2bd0d0c5883f66c699484d700a948ad183db9474dd0f872b7.
  P027-intermediate artifact 13837486719 matched
  238fa40dc10ce8c6204deb110582bcecb2249e94bc8ca36a0f5e4ee69543794b.
  Current manifests are ios-p028-migration-direct.json and
  ios-p028-migration-via.json. Prior source-43 manifests are preserved with
  -43e2b92 filenames.
- iOS Build run 42980635634: native job 130389963296 PASS, Procursus job
  130389963643 PASS, artifact-consumer verification job 130396481957 PASS.
  Runtime patching, Go 1.27.4 x509 checks, duplicate native builds,
  Mach-O/iOS 12 inspection, runtime instruction checks, disabled Wi-Fi and
  private-linkage checks, and frontend/payload reproduction all passed.
  Stable release jobs were skipped by workflow policy.
- Same-source artifacts:
  - beszel-ios-arm64, ID 13837204100, SHA-256
    a60ca0978b25d134faee0b73bc6ad3579031e03a6dc9c84866ca00b04a46f426.
  - ios-native-validation, ID 13837229020, SHA-256
    c279e8a41c5dcab4fbd3729e16807815122abdd07c356cdc29b0e0b48e5b6543.
  - ios-wifi-diagnostic-disabled, ID 13837972653, SHA-256
    f8ada151705186f635532fa1cea95bfa70d0f464d47aaf5bc34b35a2b52d48a3.
  - artifact-consumer verification manifest, ID 13837649813, SHA-256
    6073ec70531751336c2644897789920f8b8537719113c19130035298a088a1d8.
    It confirms matching independently downloaded archive digests, exact
    source/tree identity, payload hashes, ARM64 Mach-O metadata, iOS minimum
    12.0, Go build info, and release authorization status not_authorized.

Earlier scan outcomes remain historical and are not attributed to a later
source. At b8722c5d, the remaining release-owner disposition was module-only
GO-2026-5932 and portable upstream rows: number formatting, pagination and
locale catalog changes. The current state and exact next work are below.

Device qualification: DEFERRED — OWNER APPROVED. No physical device, production
database, installation, reboot, PF, LaunchDaemon, stable release, merge, or
deployment action occurred. Production iOS remains unchanged.

## Current recovery checkpoint — 2026-11-11

Repository: `/workspace/work/Beszel-iOS`. Branch `ios/p028-dependency-migration`.
The latest pushed and locally tested source commit is
`3b24fefa07cd6c34d0ecb228e84e93a70e4dd3fb`; the branch remote matched this
SHA after push. Base is P027
`c239ce0d8931405f728ef4763ad68bbbe187a9ac`. Draft PR #32 remains open and
targets `ios/p027-upstream-port`. Production `iOS` remains
`0c3e4cd22ac24264834c401ad412e44aeda4e4f3`.

New source commits since b872:

- `53dffff5507ec7a37574bf8fadb4cc1772738341` ports upstream
  `d3161e0212b0f736850daf0db73f2411143a3c9c`: number grouping and decimal
  separators now follow the active app locale, including smart cycles and chart
  tooltip values. Latin digits are retained. The formatter cache is keyed by
  locale and decimal precision. Two CPU metrics labels follow upstream invariant
  labels from `d781ced6c3c74ce4f48b8b0e500500d38d949095`.
- `3b24fefa07cd6c34d0ecb228e84e93a70e4dd3fb` adds regression coverage for
  German and English grouping/decimal separators and locale switching.

Validation on the exact source commit `3b24fefa07cd6c34d0ecb228e84e93a70e4dd3fb`:

- From `internal/site`, `PATH=/workspace/.tools/bun/node_modules/.bin:$PATH bun test`:
  PASS, 66 tests and 323 assertions.
- From `internal/site`, `PATH=/workspace/.tools/bun/node_modules/.bin:$PATH bun run build`:
  PASS. Vite reports its existing large-WASM-chunk warning and dynamic English
  locale import notice.
- `python3 scripts/audit/validate_frontend_assets.py --dist internal/site/dist --write-manifest docs/ios-frontend-assets-manifest.sha256`:
  PASS, 99 files, 3,431,298 bytes, 34 locales. Re-running with
  `--manifest docs/ios-frontend-assets-manifest.sha256` also passes; manifest
  digest `268cf3ab0eab0e52ee4cebb6d30c2e541e6917ec5858bc3881ef50b13e1fd454`.
- One `bun test` from the repository root exited 1 because that invocation
  bypassed the frontend test environment (`localStorage is not defined` in five
  suites); rerunning from `internal/site` passed all tests.
- Go, migration harness and prior security checks are unchanged by this
  frontend-only source change; the exact source b872 results remain documented
  above and must not be attributed to this new commit.

Exact-source hosted qualification for
`3b24fefa07cd6c34d0ecb228e84e93a70e4dd3fb` completed:

- P013 migration run `42986634518`: direct job `130409262419` PASS and
  P027-intermediate job `130409262927` PASS. Their independently downloaded
  artifact ZIPs match GitHub SHA-256: direct artifact `13839699109`,
  `298876dd77dd3d1b2a65b8e007d041bfc21ddfa44f551d74298f57bfbe78b94d`; via
  P027 artifact `13840695149`,
  `5d5662275578f955c3d8c485ebd497fb1a396fe90a985ea884c77a0aa32a6fd6`.
  The exact-source fixture manifests are saved as
  `ios-p028-migration-direct-3b24fefa.json` and
  `ios-p028-migration-via-3b24fefa.json`.
- iOS Build run `42986634545`: Procursus package update job `130409265496`
  PASS, native build job `130409265962` PASS and artifact-consumer verification
  job `130418095211` PASS. Release-validation and release jobs were skipped by
  workflow policy. Independently downloaded artifact ZIP digests match GitHub:
  `beszel-ios-arm64` ID `13840982221`,
  `263c9039ec22282a91ebcd23fa44ba9062474cff4e32316b91c61367b340447d`;
  `ios-native-validation` ID `13840304140`,
  `3b79ca51894cb73b76cc7b0788481011d482e7e7045e48b291ebf5dca2420745`;
  `ios-wifi-diagnostic-disabled` ID `13840987193`,
  `47e62d23ae4af09da5e2e2c933de8910dac768c0810bafa3e5e893fb86157f85`;
  `ios-artifact-consumer-verification` ID `13841445932`,
  `fb52375489dec0bb90e209a36e9f0efa1b9af87e05730be26846a13eec44c933`.
  The consumer manifest records commit `3b24fefa07cd6c34d0ecb228e84e93a70e4dd3fb`,
  tree `3da56a5604dcda9f912dd23a5bea86fdb20a40bc`, matching binary hashes,
  ARM64/iOS 12.0 metadata, Go 1.27.4 `no_self_update` build tags, runtime
  instruction checks, byte-identical isolated builds and release authorization
  `not_authorized`. It is saved as
  `ios-p028-artifact-consumer-verification-3b24fefa.json`.

Implementation source `3b24fefa07cd6c34d0ecb228e84e93a70e4dd3fb` is pushed and
remote-confirmed. The handoff and verified evidence bundle were first pushed
in docs-only commit `99312cc601b769325c6b239c406d0d34f05b8ef7`; subsequent
handoff-only updates also leave application source unchanged and do not claim
new code qualification. Historical b872 migration and native artifacts remain
valid only for their recorded source SHA.

Remaining P028 work:

- Reconcile pagination commit `237f9641960f6192ed18b42833046d8e3346911f`
  manually. Its upstream patch replaces virtualized tables and P026/P027 system
  name measurement and persisted settings; no wholesale cherry-pick was made.
- Review the full locale catalog commit `d781ced6c3c74ce4f48b8b0e500500d38d949095`.
  Only the two invariant CPU labels were ported; translations for other upstream
  strings require feature-by-feature applicability review.
- Record the release-owner disposition for GO-2026-5932. The Go 1.27.4
  symbol scan exits 0 with only that required-module advisory and no imported
  or reachable package finding; it is not a clean module audit.
- Exact-source P013 migration and iOS Build runs for `3b24fefa` completed
  successfully; their jobs, artifact IDs, archive digests and consumer
  verification are recorded above. The implementation source is pushed, while
  this handoff and its evidence manifests are local documentation changes
  awaiting commit.
- Continue by adapting upstream pagination commit
  `237f9641960f6192ed18b42833046d8e3346911f` around the P026/P027 virtualized
  table and persisted-settings behavior. Keep PR #32 Draft and based on
  `ios/p027-upstream-port`.

Device qualification: **DEFERRED — OWNER APPROVED**. No device, production
database, deployment, release, merge, PF or LaunchDaemon action occurred.
Production `iOS` remains unchanged.


## Recovery checkpoint — 2026-11-11, x/crypto security fix

Repository: `/workspace/work/Beszel-iOS`. The implementation source is
`b85cd8d9ee3bd7f054a6222fb2b50a265a3020f9` on
`ios/p028-dependency-migration`, based on P027
`c239ce0d8931405f728ef4763ad68bbbe187a9ac`. The remote branch was confirmed
at that source before this handoff-only update. PR #32 remains Draft and targets
`ios/p027-upstream-port`. Production `iOS` remains
`0c3e4cd22ac24264834c401ad412e44aeda4e4f3`. Official Beszel main remains
`800d18bbb5fa2240b6dbc7e155f3af0c34c01ad5`.

Source commit `b85cd8d9` updates `golang.org/x/crypto` from v0.58.0 to
`v0.58.1-0.20261009181718-2f1f834330b2`, upstream source commit
`2f1f834330b2d36ab3db8bfb33aaa1ac16794588`, addressing GO-2026-5932. The
existing x/net v0.62.0, x/sys v0.49.0 and x/term v0.47.0 selections remain.

Exact local validation on the source tree:

- With Go 1.27.4, `go mod verify`: PASS; `go mod tidy -diff`: PASS.
- `go test -p 2 ./agent ./internal/hub/...`: PASS.
- `go vet -p 2 ./agent ./internal/hub/...`: PASS.
- `govulncheck v1.9.0`, built with Go 1.27.4, `-json ./...`: exit 0,
  symbol-level source scan, 62 modules, x/crypto at the fixed version, zero
  findings. Raw report SHA-256:
  `981baa0cd155d54ce244bfeef24e67f52234e09f91d6be98a7e4efdd3b21204a`.
- A scanner binary built with Go 1.26.11 against Go 1.27 source exited 1 with
  the exact diagnostic: `This application uses version go1.26 of the
  source-processing packages but runs version go1.27 of 'go list'`; it also
  reported `internal/testutil/telemetry/fixtures.go:3:1: package requires
  newer Go version go1.27 (application built with go1.26)`. Rebuilding the
  pinned scanner with Go 1.27.4 resolved it.
- `govulncheck -scan=module ./...` exited 2 with
  `patterns are not accepted for module only scanning`; the supported full
  `./...` symbol-level source scan above completed successfully.

Exact-source hosted checks for `b85cd8d9ee3bd7f054a6222fb2b50a265a3020f9`:
P013 run 42998692955 PASS for direct job 130449111834 and via-P027 job
130449112349. Artifacts: direct 13845298529, SHA-256
`bba58bc39b13b4b13e80c162896c4846428ec49163dbbe84309dd7e3f8a20b0d`; via-P027
13845952510, SHA-256
`9c8b330255a84a8e73a03817fb58884fbe0071c341a8d95f474b04869cef642c`.
iOS Build run 42998693337: Procursus job 130449114387 PASS; native build job
130449114026 was QUEUED at this checkpoint. Older CI for 3b24fefa remains
historical and is not attributed to b85cd8d9.

Current P028 status: the GO-2026-5932 module finding is addressed by the fixed
upstream version and clean local symbol scan. Remaining work includes the
adapted pagination port and full locale-catalog applicability review. Next
action: complete exact-source iOS build/artifact qualification, then continue
pagination commit 237f9641960f6192ed18b42833046d8e3346911f around the preserved
virtualized table and settings behavior. Device qualification remains
**DEFERRED — OWNER APPROVED**. No physical-device or production action occurred.


## Exact-source hosted qualification completion — 2026-11-11

Implementation source `b85cd8d9ee3bd7f054a6222fb2b50a265a3020f9` remains the
only source change after the P028 frontend/migration checkpoint. PR head
`731f0032a4e9dbb1c232ca1be1e3edacafb594f6` contains that source plus only this
handoff document; GitHub comparison `b85cd8d9..731f0032` confirms only
`docs/ios-p028-handoff.md` changed.

Exact-source migration run 43001248199 on PR head 731f0032: direct and via-P027
jobs PASS. Artifacts: direct 13847380032, SHA-256
`c1673cf43baef6c69cd3f7c6ae37b789034e8b8297d656913194109478049daa`; via-P027
13846247625, SHA-256
`40174de8b1b7fe425dad5a5ecbabdefb295ea5cd14b87b5ebde4a303dd608494`.

Exact-source iOS Build run 43001248403: Procursus, native build and independent
artifact consumer all PASS. The native job passed the real frontend rebuild,
source-selection checks, portable telemetry and fault/race/network-monitor
regressions, macOS/GPU parser checks, iOS Wi-Fi adapter compatibility, Go
1.27.4 x509 security validation, isolated legacy ARM64 runtime patch, repeated
Agent/Hub builds, runtime instruction and Mach-O inspection, no-private-linkage
check, and payload/frontend manifest generation. The consumer independently
rebuilt the frozen frontend and verified same-run archives, payload bytes and
provenance against source 731f0032.

Artifacts from run 43001248403:

- `beszel-ios-arm64` 13846494954, 16,357,081 bytes, SHA-256
  `bdf8cdf68d99007f99d5fe5055247a2c65e5ebabe4c6485bcb00086a3483ed43`.
- `ios-native-validation` 13848270115, SHA-256
  `6501ac26cc7cd9bad3a919405145a940ace819b084de65a7032dcb02b951d8b4`.
- `ios-wifi-diagnostic-disabled` 13847961105, SHA-256
  `0a9ec65fac07a41ef4479d1772234aae963bc5f02ae406215bebc9975bab4e8c`.
- `ios-artifact-consumer-verification` 13846669640, SHA-256
  `ce9554bd934dcd3c310dc195a2b2956402910a5576071da768d011478871d4fb`.
  Consumer-verified Agent SHA-256:
  `e7d1bc1f92df4c8a855b429935bb6edc4817f7ce809a4ed10c2926644de69954`; Hub
  SHA-256: `d0485a3196f8ef17331cea75509570335a6e3a71e50989f7cd1f32c483a16a3d`.

The previous iOS Build run 42998693337 on source b85cd8d9 was cancelled after
a handoff-only push while its native job was still queued. Its successor
43001248403 on the docs-only head above completed PASS; no app source or CI
files changed after b85cd8d9. No release jobs were authorized; release jobs
were skipped. Device qualification remains **DEFERRED — OWNER APPROVED**.

Next P028 action: adapt upstream pagination commit
`237f9641960f6192ed18b42833046d8e3346911f` without losing P026/P027 virtualized
tables, system-name measurement and persisted settings; then review the
remaining locale catalog entries. P029 has not started.

## P028 resume checkpoint — 2026-11-12

Recovery began from the requested branch head `619c5e174bcb769cb68aebae28aa582e817deacf`; the remote branch matched before edits. The original checkout at `/workspace/Beszel-iOS` was preserved untouched because it was a separate clean `work` branch. Work continued in `/workspace/Beszel-iOS-P028` on `ios/p028-dependency-migration`, still based on P027 `c239ce0d8931405f728ef4763ad68bbbe187a9ac` and Draft PR #32.

The pagination adaptation is in `b31f1daddbcc51fc1cf2b78aa6c66ee235462a1a`, mapped to upstream `237f9641960f6192ed18b42833046d8e3346911f`. Shared page sizes are 10, 20, 50, 100 and 200. The hook stores the preference in local storage and `$userSettings`, queuing authenticated changes through the existing user-settings API. The systems, containers, network monitors, systemd, SMART, storage pools, package updates and alert history tables now use the setting. TanStack sorting and filtering remain ahead of pagination. The existing systems/network/systemd/containers/SMART virtualizers remain; their measured row output and system-name width logic are preserved, and their scroll area resets when paged rows change. Alert history retains its existing small-screen controls.

The locale review is in `0ab75a0b4fec6ed858e9ffbee35f8b05399a1793`, mapped to upstream `d781ced6c3c74ce4f48b8b0e500500d38d949095`. It ports the applicable certificate-expiry alert text, TLS certificate monitor controls, token copy/hide/reveal text, and shared pagination labels. Existing nonempty local translations were kept; missing translations were filled without replacing newer local wording. The two invariant CPU labels from this upstream change were already present. Validation found the 549 active English message keys in every locale and all 14 applicable P028 messages translated in all 34 catalogs. Lingui still reports 5–13 untranslated older messages in some catalogs; these retain the existing English fallback. The selected-locale number formatting from the earlier P028 commit remains covered by its regression test.

Local validation for implementation source `0ab75a0b4fec6ed858e9ffbee35f8b05399a1793`:

- From `internal/site`, `PATH=/workspace/.tools/bun/node_modules/.bin:$PATH bun test`: **PASS**, 74 tests / 354 assertions. Coverage includes empty, first, partial and clamped pages; sorting and filtering before pagination over 10,000 rows; page-size persistence and storage failure fallback; virtualized scroll reset; measured wide-character system-name sizing; and narrow mobile wrapping.
- From `internal/site`, `PATH=/workspace/.tools/bun/node_modules/.bin:$PATH bun run build`: **PASS**, TypeScript and Vite production build. The build emitted 34 locale bundles and the real `dist` assets embedded by `internal/site/embed.go`. Existing notices remain for the dynamic English locale import and the large WebAssembly chunk.
- `scripts/audit/validate_frontend_assets.py --dist internal/site/dist --manifest docs/ios-frontend-assets-manifest.sha256`: **PASS**, 98 files / 3,443,778 bytes / 34 locales; manifest SHA-256 `a22bf2562a522382be84b755b3bdd4ecae960f5c66c1680435c3fef9e517b21c`.
- Locale key-parity and applicable-message validation with polib: **PASS**, 34 catalogs, 549 active keys, 14 required P028 messages translated in every catalog.
- `python3 -m unittest discover -s tests/hub-migration -p 'test_*.py' -v`: **PASS**, 7 P013 backup/recovery unit tests.
- `env -u NETWORK GOTOOLCHAIN=local PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.4.linux-amd64/bin:$PATH go test -count=1 -p 1 -tags='testing no_ui no_self_update' ./...`: **PASS**. Clearing `NETWORK` is required because the container exports its desktop network name and legacy Agent tests use that variable as their default.
- The same Go 1.27.4 environment, `go vet -p 1 -tags='testing no_ui no_self_update' ./...`: **PASS**; `go mod verify`: **PASS**, all modules verified.
- Exact-source hosted P013 migration, iOS Build, artifact consumer/provenance and current security scan: **NOT RUN** at this checkpoint; they must qualify the pushed source. Do not reuse older CI as qualification for this source.

Upstream mappings are recorded in `docs/ios-p028-commit-map.csv`; both upstream ledger rows are updated. The tested frontend and Go source contains no physical-device work. Hosted qualification remains pending, so P028 is **PARTIAL — RESUMABLE**. Physical-device tests remain **DEFERRED — OWNER APPROVED**. Production `iOS` is unchanged, PR #32 remains Draft, and P029 has not started.

Next exact action: verify the origin head has not moved, push the incremental commits and updated ledger/handoff to `ios/p028-dependency-migration`, then record the exact-source P013, iOS Build and artifact/provenance results here and in Draft PR #32.
