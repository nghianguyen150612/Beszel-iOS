# Native iOS build validation — P007

The starting integration SHA is
`6b14ebcdadc44641f4f84a5d7999ebf964905aa3`; Gate B ancestry and all 178 frozen
upstream commits were verified before edits. The original workflow was recorded
at that SHA (Git blob `eae8a2f6`, full hash in JSON). P006's successful native
build is historical evidence, not validation of P007.

## Hardened contracts

| Surface | P007 behavior |
| --- | --- |
| Runtime | Strict Go 1.27.1 complete instruction match; zero-guarded YIELD; atomic write; unsupported/repeated invocation fails without mutation. |
| Toolchain | Complete job-local GOROOT copy, verified version/layout, GOTOOLCHAIN=local and dedicated iOS cache; original runtime hash unchanged. |
| CGO | GOOS=ios, GOARCH=arm64, CGO_ENABLED=1 for both binaries; GOARM64=v8.0 baseline, common device SDK clang wrapper; inherited CGO/GOFLAGS cleared. |
| SDK | `xcrun --sdk iphoneos` resolves clang and sysroot; require existing iPhoneOS device SDK/compiler; reject conflicting caller target flags; `-arch arm64 -mios-version-min=12.0`. |
| File selection | Real `go list -mod=readonly -deps -json` of both entry points for ios/darwin/linux ARM64 with CGO=1; exact platform implementation families and dependency selections. |
| Frontend | Bun 1.4.0 frozen install → sync → tests → production build; lock unchanged; real index/JS/CSS required; every generated asset's bytes must be embedded in Hub. |
| Binary | Native file/otool plus parsed Mach-O: thin ARM64 ALL MH_EXECUTE, device iOS platform 2/minos 12.0, executable entry point; legacy iPhoneOS minimum supported without accepting another platform. |
| Entry package | Actual `go version -m` must identify the correct Agent/Hub main package, Go 1.27.1, ios/arm64/CGO=1, ARMv8.0 and clean Git revision matching the tested HEAD. |
| Assembly | Separate unstripped linked Agent and Hub validation builds; exact procyield opcode and branch checks; production stripping retained. |
| Payload | Exactly Agent, Hub, SHA256SUMS; two unique bare-filename hashes; nonempty regular binaries; both checksums verified. Validation JSON is a separate review artifact. |
| Repeat build | Two builds per binary using independent clean caches; compare bytes in the same job. |

The previous Mach-O check accepted any LC_BUILD_VERSION with minos 12.0, even a
macOS or simulator platform. The new parser checks platform and deployment
version together, rejects ARM64e/other CPU targets and invalid executable entry
points, and has deliberately malformed binary/payload fixtures.

One test selection defect was established: `gpu_darwin_test.go` used `darwin`
alone and selected unavailable macOS parser symbols on iOS. It now uses
`darwin && !ios`, matching the production collector. No production collector or
concurrency code changed; no additional race gate is required for these tooling
and test-constraint changes.

## Platform findings and limits

- iOS selects battery_ios.go and system_platform_ios.go; excludes macOS battery,
  Wi-Fi AppleScript and GPU collectors; selects Wi-Fi/GPU unsupported fallbacks.
  CPU/memory/network cgroup implementations are excluded. Each known platform
  family has exactly one implementation.
- macOS retains battery_darwin.go, wifi_darwin.go and gpu_darwin.go, and excludes
  iOS metadata/battery. Hosted CI runs GPU parser tests and both Darwin builds
  with the original unpatched Go toolchain.
- Linux retains its battery, Wi-Fi, CPU/memory/network-counter and cgroup files,
  excludes iOS files and builds both entry points locally.
- gopsutil v4.26.9 intentionally selects Darwin CPU/memory/disk/net/host/load and
  ARM64 sensor files on iOS. Their actual constraints were inspected. Darwin
  selection is not proof of iOS runtime availability: IOKit/CoreFoundation,
  sensors, disk I/O and syscall/command behavior still require P008/P010/P018.
  The existing iOS guard skips the Darwin CPU model probe. No new metrics,
  network-monitor, package-manager or Wi-Fi feature is implemented here.

## Executed regression

The matrix/readiness validators pass (178 upstream SHAs, 13 logical patches).
All 49 audit tests pass; 35 cover P007 runtime/build contracts, including 15
source mutations and malformed CPU/platform/deployment/checksum/asset fixtures.
Existing shell suites pass: upstream sync 10, iOS sentinels 44, bootstrap 54,
installer 814. `git diff --check` and actionlint pass.

The canonical `env -u NETWORK go test -tags 'testing no_ui' ./...` passes; a second
full run with `-count=1` passes without cached test results. Focused Agent,
battery and Wi-Fi tests and both Linux builds pass. Bun tests: 44 passed,
0 failed; lock SHA-256 remains
`16643e3228f9b0fac1e5134d89782c096f49e8071489cc2a4f977f3565f02afa`.
All 95 local production asset files were found byte-for-byte in the actual Linux
Hub. Native CI independently checks its own generated assets in the iOS Hub.

The first baseline Go run failed the existing timing-sensitive
`TestConnectionManager_ReconnectionLogic`: a reconnect goroutine can reset
isConnecting before the immediate assertion observes it. The test and production
connection code are unchanged from the starting SHA; the test's last commit is
upstream `4ffd83677d18fe285ce1647da9d64ed4b2798f40`. Subsequent canonical and
uncached suites pass. This known scheduling-sensitive test is recorded for
P009/P017; it was not weakened or changed to obtain a pass.

## Hosted evidence

Exact runs, tested commit, versions, runtime hashes, source selection, native
metadata and actual artifact checksums are in
[`ios-build-validation.json`](ios-build-validation.json). Implementation and
repair commits are `7b09aa797fbfa636e05c67e85f7df307e474e237` and
`19c705c65a45b3da1c426e3c0222636fd244f3b0`. Final provenance guards are in
`d85eb2cc6f139d7e03c98e8f793eddf193963d4d`; their hosted result is checked on the
final HEAD recorded in the PR.

The first native run
[38500460499](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38500460499)
built both binaries, passed both clean-cache comparisons, disassembly and native
platform checks, but failed the new frontend validator's incorrect assumption
that Vite URLs start with `/assets/`. Beszel's real Vite base is `./`. The repair
accepts and tests `./assets/` references, complete embedding and missing assets;
the failed run is not a P007 pass.

[PR #9](https://github.com/nghianguyen150612/Beszel-iOS/pull/9) records the final
feature HEAD, its authoritative iOS/installer reruns, exact hashes and merge
checks. Evidence files record the tested source candidate; the final commit
cannot embed its own hash. Changed build inputs require a new hosted run.

Identical outputs from independent caches demonstrate byte reproducibility for
the stated commit/SDK/toolchain/frontend inputs within that hosted environment.
Cross-commit hashes include VCS revision metadata and need not match; different
SDKs, runners or future versions are not covered by that result.

No signing command, release/tag creation, deployment, device installation or
production database modification occurred. Native build artifacts are for
review only. A7-DEVICE-01 stays P018 / Gate F; historical 0.19 iPad records remain
unchanged. P007 does not complete Gate C or start P008.
