# Beszel-iOS Port Status

Status labels used here: **Working** (implemented and believed correct), **Tested** (observed on the validated device), **Experimental** (present but lightly validated), **Untested** (no evidence), **Planned** (not implemented).

## Purpose

Provide native Beszel Agent + Hub binaries for jailbroken iOS devices while preserving upstream Beszel functionality. This is an unofficial community port, not a fork of the monitoring model.

## Upstream relationship

- Upstream: [henrygd/beszel](https://github.com/henrygd/beszel).
- `main` is the preserved upstream-aligned branch; it is not advanced as part
  of iOS maintenance work.
- `iOS` = upstream base + Beszel-iOS-specific files/changes (see below). No Agent/Hub rewrite, no Hub database changes, no removed upstream features.
- The iOS base is recorded from `beszel.Version`, while untagged upstream-main
  commits are audited separately before integration. See
  [docs/upstream-sync.md](upstream-sync.md) for the current snapshot,
  classifications, and procedure.

## Branch model

- `main` — upstream-aligned. No iOS-only changes.
- `iOS` — active Beszel-iOS port branch and repository default branch. All iOS work belongs here.

## Validated hardware

**Tested:**

- iPad mini 2 (iPad4,4 / A1489), Apple A7, arm64
- iOS 12.5.7, jailbroken, semi-untethered Amethyst / Procursus environment

After a full reboot, manual jailbreak reactivation is required before the
custom LaunchDaemons can operate; after reactivation, Agent and Hub were
verified to return automatically. Stock/non-jailbroken iOS is not supported.

Everything else is **Untested** until a real device report lands in the repo.

## iOS-specific implementation (verified present)

| Path | Purpose | Status |
| --- | --- | --- |
| `agent/battery/battery_ios.go` (`//go:build ios`) | iOS battery via `AppleARMPMUCharger` + `ioreg -l`; parses Current/Max/RawMax capacity, ExternalConnected, IsCharging, FullyCharged, BatteryInstalled; exposes `Primary` battery | **Tested** — validated shape `Battery:[23 3]`, `Batteries:map[Primary:23]` |
| `agent/battery/battery_darwin.go` (`darwin && !ios`) | Restricts macOS `AppleSmartBattery -a` path to non-iOS so iOS uses the charger class instead | **Working** |
| `agent/system.go` (`runtime.GOOS != "ios"` guard + `adjustPlatformSystemDetails()` hook) | Avoids gopsutil Darwin CPU probe on iOS; applies iOS metadata overrides | **Working** |
| `agent/system_platform_ios.go` (`//go:build ios`) | Sets OS/Arch, hostname/kernel/cores/threads via sysctl, CPU model from `hw.machine` (A7 family mapping), `OsName` from `SystemVersion.plist` | **Tested** |
| `agent/system_platform_other.go` (`//go:build !ios`) | No-op hook so non-iOS builds are unchanged | **Working** |
| `.github/scripts/patch-go-ios-arm64-runtime.py` | Replaces `CNTVCT_EL0`-based `procyieldAsm` with legacy `YIELD` loop; refuses to patch unknown runtimes | **Tested** (required on A7/iOS 12) |
| `.github/workflows/ios-build.yml` | Consolidated pipeline: one macOS job builds Agent + Hub into `build/ios/`, verifies Mach-O/iOS-12 metadata, generates + verifies `SHA256SUMS`, uploads single `beszel-ios-arm64` artifact; a dependent `release-ios` job (tag pushes only) re-verifies the payload and publishes exactly those three files as a non-draft, non-prerelease GitHub Release marked Latest | **Working** |

Verified build-tag selection: `GOOS=ios go list ./agent/battery` yields only `battery.go + battery_ios.go`; `GOOS=darwin` yields `battery_darwin.go`. `gofmt` clean, `go test ./agent/battery` passes on Linux, `go vet` passes for the iOS battery package.

## Component status

### Agent — Tested

- Binary: `/usr/local/bin/beszel-agent`, data dir `/var/lib/beszel-agent`, port `45876`.
- LaunchDaemon: `/Library/LaunchDaemons/dev.beszel.agent.plist`.
- Reports system metrics over the standard Beszel Agent protocol to the Hub.
- CPU/memory/disk/network/load metrics flow through shared upstream code (gopsutil-based); only the CPU-model probe and metadata are iOS-specific.

### Hub — Tested

- Binary: `/usr/local/bin/beszel-hub`, data dir `/var/lib/beszel-hub`, port `8090`.
- LaunchDaemon: `/Library/LaunchDaemons/dev.beszel.hub.plist`.
- Health: `http://127.0.0.1:8090/api/health`.
- **`/var/lib/beszel-hub` is never disposable** — it holds user DB / config. Upgrades and packaging must preserve it.

### Battery — Tested

- Uses `/usr/sbin/ioreg -r -c AppleARMPMUCharger -l` (not `-a`; iOS 12 `ioreg` errors on `-a`).
- Handles legacy `| "`-prefixed lines and skips the nested `BatteryData` dict by matching top-level `"Key" = ` prefixes.
- Percentage + charging/discharging/full/idle/empty mapping validated on hardware.

### System metadata — Tested

- Hostname (`kern.hostname`), kernel (`kern.osrelease`), cores/threads (`hw.physicalcpu`/`hw.logicalcpu`), CPU model from `hw.machine`, OS name `iOS <ProductVersion>`.
- `iPad4,*` → `Apple A7`; `iPhone6,1/6,2` → `Apple A7`; otherwise `Apple SoC (<machine>)`.

### Network / filesystem metrics

- **Working** via shared upstream Agent code (no iOS fork of those paths). No iOS-specific regressions observed on the validated device, but per-interface and per-mount coverage on iOS is **Experimental** — treat edge cases (VPN interfaces, iOS mount layout) as unvalidated.

### Apple A7 Go runtime workaround — Tested

- Modern `procyieldAsm` used `CNTVCT_EL0`, unusable in the tested legacy iOS userspace → `SIGILL` in `runtime.procyieldAsm`.
- The Python patch is applied with `sudo` on the macOS runner before both iOS builds. It is load-bearing; do not delete or "simplify".

### Build status — Working

- Consolidated `.github/workflows/ios-build.yml` builds both binaries into `build/ios/`, validates Mach-O arm64 + iOS load commands + 12.0 deployment metadata, generates `SHA256SUMS` via `shasum -a 256`, verifies it, and uploads one `beszel-ios-arm64` artifact (`beszel-agent-ios-arm64`, `beszel-hub-ios-arm64`, `SHA256SUMS`).
- Tag pushes matching `v*-ios.*` additionally run the `release-ios` job, which re-downloads the artifact, re-verifies it (existence, non-zero size, exact two-entry `SHA256SUMS`, checksum match), validates the tag against `beszel.Version` and `iOS` history, then publishes the three files via `gh release create --latest` (non-draft, non-prerelease).
- The legacy probe workflows (`ios-agent-probe.yml`, `ios-hub-probe.yml`) were removed once the consolidated pipeline proved itself; all of their behavior is covered above.

## Known limitations

- Only the iPad mini 2 / A7 / iOS 12.5.7 target is validated.
- Binaries must be installed under `/usr/local/bin` with `chown root:wheel`, `chmod 755`, `ldid -S`. `$HOME`/`/tmp` execution has previously failed.
- GitHub Releases are published from iOS tags and `install.sh` covers fresh installs, transactional updates with backup/rollback, diagnostics, repair, reconfiguration, and safe uninstall with optional explicit data purge. The installer lifecycle is implemented and has been **validated end-to-end on the reference device** (see [device-validation.md](device-validation.md)), including a full reboot validation: after reboot plus manual semi-untethered jailbreak reactivation, both LaunchDaemons returned automatically and Agent reconnect plus Hub health plus data/history persistence were re-verified. No stock/non-jailbroken iOS compatibility is claimed.
- LaunchDaemon plists and packaging scripts are not yet in the repo (only documented paths; the installer generates the plists on-device).

## Untested devices / iOS versions

All other iPhones/iPads, all other SoCs (A8+), and all other iOS versions (including modern iOS) are **Untested**. In particular, do not assume the A7 runtime patch is needed — or harmless — on newer devices without testing.

## Planned installer / release work

**Existing:** consolidated CI build pipeline producing `beszel-agent-ios-arm64`, `beszel-hub-ios-arm64`, `SHA256SUMS` as a single `beszel-ios-arm64` artifact; tag-triggered GitHub Release publication of those exact three asset names (see [ios-build-notes.md](ios-build-notes.md) for the `v<upstream>-ios.<rev>` scheme); interactive `install.sh` one-line installer (v1.0.0) for fresh Agent / Hub / Agent+Hub installs with checksum verification, `ldid` signing, LaunchDaemon setup and Hub health checks, plus transactional Agent / Hub / Agent+Hub updates with release state tracking (`/var/lib/beszel-ios/install-state`), signed staging, binary backups (`/usr/local/bin/*.bak`), automatic rollback, plist preservation, and Hub database preservation, plus read-only diagnostics, conservative repair (restart in place, missing-binary restore, confirmed config recreation) and safe reconfiguration (Agent key/port, Hub port) with validated plist transactions, plist backups (`/Library/LaunchDaemons/*.plist.bak`) and automatic config rollback, plus transactional application uninstall (Agent, Hub, or Agent+Hub; data preserved by default, offline-capable) with optional typed-confirmation data purge (`DELETE AGENT DATA` / `DELETE HUB DATA`).

**Planned** (not implemented): LaunchDaemon plists under `packaging/launchd/` (currently generated by the installer instead). The installer resolves the Latest release tag once per run and pins all downloads to that tag (`.../releases/download/<tag>/...`); its raw URL targets the `ios` branch (`.../ios/install.sh`). See [architecture.md](architecture.md).
