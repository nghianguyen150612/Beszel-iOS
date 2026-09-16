# Beszel iOS

**An unofficial community port of [Beszel](https://github.com/henrygd/beszel) for jailbroken iOS devices.**

This repository preserves the original Beszel Agent + Hub architecture and makes both components run natively on jailbroken iOS. Both the Agent and the Hub have been demonstrated on real iOS hardware.

> Upstream project: **Beszel by henrygd** — <https://github.com/henrygd/beszel>
> This is a community port. It is not affiliated with or endorsed by the upstream project.

## What is Beszel iOS?

[Beszel](https://github.com/henrygd/beszel) is a lightweight server monitoring platform with a Hub (dashboard) and an Agent (per-machine metrics reporter).

Beszel iOS ports both components to jailbroken iOS with minimal divergence from upstream:

- Same Agent + Hub architecture and protocol.
- iOS-specific system metadata, battery telemetry, and build support.
- A build-time workaround for the legacy Apple A7 Go runtime issue (see below).

## Why this port exists

Stock Beszel targets Linux / macOS / Windows hosts. Jailbroken iOS devices can act as small always-on servers, but they need:

- `GOOS=ios` builds with the iPhoneOS SDK,
- iOS system identification (no macOS-only CPU probing),
- battery telemetry from the iOS I/O registry,
- a legacy ARM64 runtime workaround for older Apple SoCs.

This branch collects those changes in one place so the port stays recognizable as Beszel.

## Current status

| Component | Status |
| --- | --- |
| Agent on iOS | **Tested** — runs natively, reports to Hub |
| Hub on iOS | **Tested** — runs natively, serves UI and `/api/health` |
| Battery monitoring | **Tested** — percentage + charging state on validated hardware |
| System metadata | **Tested** — hostname, kernel, CPU model, iOS version |
| Consolidated build pipeline | **Working** — one workflow builds Agent + Hub + SHA256SUMS |
| GitHub Releases | **Working** — published from iOS tags (`v<upstream>-ios.<rev>`) |
| One-line installer | **Working** — fresh Agent / Hub / Agent+Hub installs |

See [docs/ios-port-status.md](docs/ios-port-status.md) for the full audit.

## Tested hardware

**Validated (real device):**

- iPad mini 2 (iPad4,4 / A1489)
- Apple A7, arm64
- iOS 12.5.7, jailbroken (Procursus bootstrap)

Other devices and iOS versions are **untested**. Do not assume broader compatibility. If you test another device, please report it (see Contributing below).

## Features currently working

### Agent

- Runs natively as `/usr/local/bin/beszel-agent`.
- Default port `45876`.
- Managed via LaunchDaemon at `/Library/LaunchDaemons/dev.beszel.agent.plist`.
- Reports CPU, memory, disk, network, load average, and battery to the Hub.
- Skips the incompatible Darwin CPU probe on iOS; uses iOS sysctls instead.

### Hub

- Runs natively as `/usr/local/bin/beszel-hub`.
- Default port `8090`.
- Managed via LaunchDaemon at `/Library/LaunchDaemons/dev.beszel.hub.plist`.
- Health endpoint: `http://127.0.0.1:8090/api/health`.
- Web frontend is built (`bun install && bun run build` in `internal/site`) before compiling the Hub binary.

> **Warning:** `/var/lib/beszel-hub` contains Hub database / account / configuration state. Never delete it during upgrades or packaging work.

### Battery monitoring

- Reads `/usr/sbin/ioreg -r -c AppleARMPMUCharger -l` (plain-text `-l` output).
- Do **not** use `ioreg ... -a` here: the tested iOS 12 `ioreg` fails with `can't open file` for that form.
- Parses `CurrentCapacity`, `MaxCapacity`, `AppleRawMaxCapacity`, `ExternalConnected`, `IsCharging`, `FullyCharged`, and `BatteryInstalled`.
- Validated result shape: `Battery:[23 3]`, `Batteries:map[Primary:23]`.
- Implementation: `agent/battery/battery_ios.go` (`//go:build ios`).

### iOS-specific runtime workaround

On the validated Apple A7 / iOS 12 target, the modern Go ARM64 `runtime.procyieldAsm` path (using `CNTVCT_EL0`) faults with `SIGILL`. The build applies:

- `.github/scripts/patch-go-ios-arm64-runtime.py`

which replaces that path with a legacy `YIELD` loop. This patch is **required** for the current A7/iOS 12 build. Do not remove it without validating on the same hardware.

## Installation

On a jailbroken iOS device, run:

```sh
curl -fsSL \
  https://raw.githubusercontent.com/nghianguyen150612/beszel-ios/ios/install.sh \
  | sudo sh
```

This interactive installer (`install.sh`, POSIX `/bin/sh`, installer v0.4.0)
downloads the Latest release assets, verifies their SHA256 checksums,
installs the binaries into `/usr/local/bin` (with `chown root:wheel`,
`chmod 755`, `ldid -S`), creates the data directories and LaunchDaemon
plists, starts the services, and verifies Hub health. No clone, Go toolchain,
or manual signing/plist work is needed.

Menu:

- Install Agent
- Install Hub
- Install Agent + Hub
- Update
- Repair / Reconfigure
- Uninstall
- Exit

First run installs Agent / Hub / Both. Later, re-run the SAME one-line
command and choose Update, Repair / Reconfigure, or Uninstall — no second
script is needed. After an action completes you return to the main menu;
Back returns there as well. Update resolves the current Latest release tag
once, then downloads SHA256SUMS and binaries from that same pinned release,
so a release cannot change mid-transaction. Uninstall needs no network and
no `ldid`: it works offline using only local tools.

Working:

- Agent fresh install (Hub public key + port prompts, launchd setup, service start)
- Hub fresh install (port prompt, launchd setup, service start, `/api/health` verification)
- Agent + Hub fresh install (Hub first, then Agent key flow)
- release checksum verification before anything is installed
- `ldid` signing (offers `apt-get install -y ldid` when missing; never upgrades the system)
- LaunchDaemon setup and service startup
- existing-install detection
- release state tracking (`/var/lib/beszel-ios/install-state`)
- Agent update, Hub update, Agent + Hub update (Hub first, then Agent)
- signed binary staging before any downtime
- binary backup (`/usr/local/bin/beszel-agent.bak`, `/usr/local/bin/beszel-hub.bak`)
- automatic binary rollback when a new version fails to start
- config preservation (plists are never regenerated during update; no key prompt)
- Hub database preservation (`/var/lib/beszel-hub` is never deleted, reset, or re-owned)
- diagnostics (read-only Agent/Hub status: binary, plist, service PID, ports, release; the Agent key value is never printed, only whether one is configured)
- repair (restores missing/broken service components while preserving configuration and data where possible: restart in place, restore a missing binary from Latest without re-asking config, recreate missing config only with explicit confirmation)
- reconfigure Agent (change Hub public key and/or Agent port via a validated, backed-up plist transaction with automatic config rollback)
- reconfigure Hub (change Hub listening port, health-checked on the new port, with rollback to the previous port)
- plist backups (`/Library/LaunchDaemons/dev.beszel.agent.plist.bak`, `/Library/LaunchDaemons/dev.beszel.hub.plist.bak`)
- safe uninstall (Agent, Hub, or both; transactional service removal with rollback, works offline)
- optional explicit data purge (typed confirmation only)

The installer lifecycle is complete: install, update, repair/reconfigure,
and uninstall are all covered.

### Uninstall vs purge

A normal uninstall removes only application/service artifacts:

- the service is unloaded first (never deleted from under a running service)
- `/usr/local/bin/beszel-agent` (or `beszel-hub`)
- `/Library/LaunchDaemons/dev.beszel.agent.plist` (or Hub equivalent)
- the component's binary `.bak` and plist `.bak` backups
- the component's exact service log files
- the component's installer release state

It always preserves user data:

- `/var/lib/beszel-agent`
- `/var/lib/beszel-hub`

Deleting data is a separate optional purge step offered afterwards (and also
later for already-removed applications with retained data). Keeping data is
the default. Purging requires typing an exact phrase:

- Agent data: `DELETE AGENT DATA`
- Hub data: `DELETE HUB DATA`

Anything else keeps the data. There is no shortcut to delete everything at
once. A later fresh install reuses a preserved data directory as-is: it is
never wiped, re-owned recursively, or re-created.

> **Warning:** `/var/lib/beszel-hub` contains Hub database / account / configuration state. It is preserved by install, update, repair, reconfigure, and normal uninstall alike. Do not hand-edit plists or install-state, and do not run manual `rm -rf` commands against these paths; use the installer menus.

### Manual download

GitHub Releases publish exactly three assets per iOS release:

- `beszel-agent-ios-arm64`
- `beszel-hub-ios-arm64`
- `SHA256SUMS`

Download the latest release:

```sh
curl -fsSLO https://github.com/nghianguyen150612/beszel-ios/releases/latest/download/beszel-agent-ios-arm64
curl -fsSLO https://github.com/nghianguyen150612/beszel-ios/releases/latest/download/beszel-hub-ios-arm64
curl -fsSLO https://github.com/nghianguyen150612/beszel-ios/releases/latest/download/SHA256SUMS
shasum -a 256 -c SHA256SUMS
```

Without the installer, deployment is manual: copy the verified binaries to
`/usr/local/bin` on device, sign with `ldid`, and install the LaunchDaemon
plists. Custom binaries must live under `/usr/local/bin`; running them from
`$HOME` or `/tmp` has previously caused iOS execution/sandbox problems.

## Building

iOS builds run on the macOS GitHub Actions runners:

- `GOOS=ios`, `GOARCH=arm64`, `CGO_ENABLED=1`
- iPhoneOS SDK + Apple clang wrapper, `-mios-version-min=12.0`
- Go runtime patch applied first
- Hub web UI (`bun install && bun run build` in `internal/site`) built before the Hub binary

Consolidated pipeline (manual dispatch, also runs on `ios` pushes touching build/iOS files, and on every `v*-ios.*` tag push):

- `.github/workflows/ios-build.yml` — builds Agent + Hub, verifies Mach-O outputs, generates and verifies `SHA256SUMS`, uploads one `beszel-ios-arm64` artifact containing exactly `beszel-agent-ios-arm64`, `beszel-hub-ios-arm64`, `SHA256SUMS` (under `build/ios/`).

Releases are published from version tags, not from branch pushes:

- Tag format: `v<upstream-version>-ios.<revision>` (for example `v0.19.0-ios.1`).
- Pushing a valid tag rebuilds both binaries and publishes the same three files as GitHub Release assets, marked as the repository's Latest release.
- The tag must match the `beszel.Version` declared in `beszel.go` and must reference a commit in `ios` history; otherwise the release job fails before publishing.
- Upstream release automation (`release.yml`, `docker-images.yml`) explicitly ignores `v*-ios.*` tags, so iOS releases never trigger GoReleaser or Docker image builds.

See [docs/ios-build-notes.md](docs/ios-build-notes.md) for details.

## Project branch model

- `main` — upstream-aligned branch. Do not put iOS-only changes here.
- `ios` — **active iOS port branch and repository default branch.** All iOS development belongs here.

This task, and all future iOS work, must stay on `ios`.

## Upstream relationship

- Based on [henrygd/beszel](https://github.com/henrygd/beszel).
- `main` tracks upstream; `ios` adds iOS compatibility on top.
- Upstream features, Hub database semantics, and Agent architecture are intentionally preserved.
- Frontend lag of a commit or two behind upstream is expected; iOS-only changes are the battery, system-metadata, runtime-patch, and build/release files documented in [docs/ios-port-status.md](docs/ios-port-status.md).

## Contributing / testing other devices

Helpful contributions:

- Test reports from other jailbroken devices/iOS versions (model, SoC, iOS version, jailbreak/bootstrap, what worked).
- Build logs from the iOS workflows.
- Docs fixes that keep the port recognizable as Beszel.

Please target the `ios` branch, keep upstream attribution intact, and do not remove the battery implementation or the A7 runtime workaround.

## License / attribution

MIT License — see [LICENSE](LICENSE). Original copyright retained:

> Copyright (c) 2024 henrygd

Beszel is by [henrygd](https://github.com/henrygd/beszel). This repository adds an unofficial community iOS port on the `ios` branch.
