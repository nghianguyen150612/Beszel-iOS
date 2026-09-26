# Beszel-iOS

Run Beszel Agent and Hub natively on a jailbroken iPhone or iPad — turn old iOS hardware into a lightweight monitoring node or self-hosted server.

[![License: MIT](https://img.shields.io/github/license/nghianguyen150612/Beszel-iOS)](LICENSE)
![Beszel base 0.19.0](https://img.shields.io/badge/Beszel%20base-0.19.0-blue)
![Tested on iPad mini 2 · iOS 12.5.7](https://img.shields.io/badge/tested-iPad%20mini%202%20%C2%B7%20iOS%2012.5.7-green)

[Beszel](https://github.com/henrygd/beszel) is a lightweight, self-hosted server monitoring platform. You install a small **Agent** on each machine you want to watch, and a **Hub** collects the numbers and shows them in a web dashboard — CPU, memory, disk, network, and more. This project is an **unofficial community port** that runs both the Agent and the Hub directly on jailbroken iOS, with no Linux VM or container needed.

> Upstream project: **Beszel by henrygd** — <https://github.com/henrygd/beszel>
>
> This is a community port. It is not affiliated with or endorsed by the upstream project.

## Why Beszel-iOS?

Have an old iPhone or iPad sitting in a drawer? If it's jailbroken, it can still be useful.

Beszel-iOS lets you reuse that device as a small always-on machine: run the monitoring Agent on it, check it from your normal Beszel dashboard, and — if you want — host the dashboard itself on the same iPhone or iPad. Everything runs as native arm64 iOS binaries and starts automatically with the system once your jailbreak environment is active.

In short: old iPad in, live system graphs out.

## Features

- **Native Beszel Agent for iOS** — reports CPU, memory, disk, network, load, and system info to any Beszel Hub.
- **Native Beszel Hub for iOS** — serves the familiar Beszel web dashboard and stores history right on the device.
- **Battery telemetry** — shows battery percentage and charging state alongside the usual server metrics.
- **Starts automatically** — Agent and Hub run as system services and come back after jailbreak reactivation.
- **Hub + Agent on one device** — one iPhone or iPad can monitor itself and show its own dashboard.
- **One-command installer and CLI** — install, update, repair, reconfigure, uninstall, inspect health, and control the existing services from the menu or persistent `beszel-ios` command. Downloads are checksum-verified, updates keep a backup with automatic rollback, and your data is preserved.
- **Safe by default** — normal uninstall removes the app but keeps your data; wiping data always asks for explicit confirmation.
- **Stays close to upstream** — same Agent + Hub design and protocol as regular Beszel, based on upstream 0.19.0.

## Quick Start

> **Before you start:** make sure your device meets the [Prerequisites](#prerequisites) and [Compatibility](#compatibility) below. In particular, you need a jailbroken device with SSH or terminal access.

SSH into your jailbroken iPhone or iPad, then run:

```sh
curl -fsSL https://raw.githubusercontent.com/nghianguyen150612/Beszel-iOS/iOS/install.sh | sudo sh
```

That's it — one command, no cloning or extra downloads. Then:

1. Choose **Agent**, **Hub**, or **Agent + Hub** from the menu.
2. Follow the prompts (for the Agent you'll need your Hub's public key; for the Hub you'll pick a port).
3. If you installed a Hub, open its web address to create your account. If you installed an Agent, add it to your Hub as usual.

After installation, you can open the menu or inspect the local manager and component versions with:

```sh
sudo beszel-ios
beszel-ios help
beszel-ios version
```

The curl installer remains available if you need to install or recover the manager again.

## Installer architecture

The one-line command is a **pipe-safe bootstrap**, not the installer itself:

```text
curl .../iOS/install.sh | sudo sh
        |
        v
small bootstrap (install.sh)
        |  1. downloads scripts/ios/install-beszel.sh over HTTPS
        |  2. verifies it against the SHA-256 pinned in install.sh
        |  3. executes the verified local copy as manager.sh
        v
lifecycle engine (menu, install/update, repair, reconfigure,
uninstall/purge, persistent beszel-ios CLI, status/diagnostics/doctor,
service control)
        |
        v
that exact verified engine file is persisted as
/var/lib/beszel-ios/manager.sh
```

What this provides:

- **Deterministic engine selection** for a given fetched bootstrap — the engine that runs is fixed by `engine_sha` inside `install.sh`.
- **Fail-closed TOCTOU protection** between fetching the bootstrap and fetching the engine: if the `iOS` branch changes in between, the checksums no longer match and the run aborts *before* the engine executes.
- **Verified manager persistence** — a successful install/update persists byte-for-byte the same local engine file that performed the transaction. The engine never re-downloads manager source from the branch, so a normal `beszel-ios update` refreshes Beszel application binaries only and never silently swaps the manager for a newer one.

What it is **not**: the bootstrap itself is still fetched over HTTPS from the mutable `iOS` branch, so this is a checksum-pinned install path, not full cryptographic release signing. CI (`.github/workflows/ios-installer.yml`) fails whenever the engine changes without `engine_sha` being updated in the same commit.

To adopt a newer manager after the `iOS` branch publishes one, re-run the same bootstrap command shown above — it fetches a newer bootstrap whose pinned digest corresponds to the newer engine. There is no `self-update` command, and application updates are deliberately decoupled from manager updates.

Validation status, stated precisely: the existing application lifecycle operations are [physically validated](docs/device-validation.md) on the reference iPad; the persistent CLI `service`/`doctor` commands are fixture-tested only; the new bootstrap/checksum-pin architecture is host/fixture/CI validated (`tests/bootstrap-test.sh`, `tests/install-sh-test.sh`, installer workflow) and has not been re-run on a physical device.

## CLI administration

After the persistent `beszel-ios` command is installed, use it to manage the application and its existing LaunchDaemons:

```sh
beszel-ios help
beszel-ios version

sudo beszel-ios install agent
sudo beszel-ios install hub
sudo beszel-ios install both
sudo beszel-ios update both

beszel-ios status
beszel-ios diagnostics
beszel-ios doctor

sudo beszel-ios repair agent
sudo beszel-ios reconfigure hub

sudo beszel-ios service hub restart
sudo beszel-ios service agent restart
beszel-ios service both status
sudo beszel-ios service both restart

sudo beszel-ios uninstall agent
sudo beszel-ios uninstall hub
sudo beszel-ios uninstall both
```

`status` gives concise component and runtime state. `diagnostics` shows detailed, read-only low-level observations. `doctor` provides a read-only health assessment with `PASS`, `WARN`, or `FAIL`; a warning alone does not make the command fail. Bare `doctor` checks installed components, while `doctor agent`, `doctor hub`, and `doctor both` explicitly require the selected components to be installed. Exit codes are `0` when there are no failures, `1` when a check fails, and `2` for invalid CLI syntax. None of these commands prints the Agent key.

`service ... status` is read-only. Service `start`, `stop`, and `restart` require root. They operate on the LaunchDaemon plists created by application install; the LaunchDaemons remain the only persistent supervisors. For `both`, dependency order is:

```text
start:   Hub -> Agent
stop:    Agent -> Hub
restart: Agent stop -> Hub stop -> Hub start -> Agent start
```

Normal uninstall removes application and service files but preserves Agent and Hub data. To request the separate destructive purge path, use:

```sh
sudo beszel-ios uninstall hub --purge
```

The purge path still requires the exact typed confirmation. Deleting the Hub database, accounts, and history is irreversible without an external backup. The completed service and doctor CLI has fixture-test coverage but has not been validated on a physical iPad.

## Prerequisites

- A jailbroken **arm64** iPhone, iPad, or iPod touch.
- A working jailbreak environment with standard Unix tools (`curl`, `launchctl`), network access, and a SHA-256 tool (`sha256sum`, `shasum`, or `openssl`).
- A terminal on the device, or SSH access to it.
- The ability to run commands as root (`sudo`) for installation.

The validated setup is an **Amethyst + Procursus** environment (see [Compatibility](#compatibility)). Other jailbreak setups may work, but they haven't received the same real-device testing yet. The installer will offer to install `ldid` (used to sign iOS binaries) if it's missing.

## Compatibility

This section uses three distinct statuses. They mean different things:

- **Validated** — actually tested on a real device by this project.
- **Expected compatible (not yet validated)** — technically plausible from the build and installer design, but not yet tested on real hardware.
- **Not supported** — not expected to work with the current installer and binaries.

### Validated hardware

Only one configuration has completed the full real-device test matrix (Agent install, Hub install, Agent + Hub together, diagnostics, repair, reconfiguration, safe uninstall, reinstall with retained data, reconnect, battery telemetry, reboot, jailbreak reactivation, and automatic service return):

| Device | Model | SoC | OS | Jailbreak / Bootstrap | Status |
| --- | --- | --- | --- | --- | --- |
| iPad mini 2 | iPad4,4 / A1489 | Apple A7 | iOS 12.5.7 | Amethyst + Procursus (semi-untethered) | **Validated** |

Full evidence is recorded in [Real-device validation](docs/device-validation.md).

### Architecture and OS target

The current binaries are built for:

- Architecture: **arm64** (`GOOS=ios`, `GOARCH=arm64`).
- Minimum iOS deployment target: **iOS 12.0** (the build passes `-mios-version-min=12.0` and verifies the iOS deployment metadata in the binaries).

In plain language: the binaries are *built for* arm64 with an iOS 12.0 minimum target. Other jailbroken arm64 devices meeting that target *may therefore be compatible*, but they have **not** received the same real-device validation yet. A minimum deployment target is not a support promise for every later iOS release or every newer chip — it only describes what the binaries were built against.

See [Building iOS binaries](docs/ios-build-notes.md) for the exact build configuration.

### Which devices can I try?

#### Tested

- The iPad mini 2 setup in the table above. If your device matches it exactly, you are on the validated path.

#### Likely / expected to be compatible (not yet validated)

Based on what the build and installer actually require, the following *may* work but are **not yet real-device validated**:

- A jailbroken arm64 iPhone, iPad, or iPod touch.
- Running an iOS version at or above the binaries' minimum deployment target (iOS 12.0).
- Able to run native `ldid`-signed arm64 binaries outside the App Store sandbox.
- Able to provide root / `sudo` access for installation.
- Supporting `launchd` / LaunchDaemons via `launchctl`.
- Providing a writable jailbreak environment at the traditional paths the installer uses (see below).
- Providing working `curl`, network access to GitHub releases, a SHA-256 tool, and `ldid` (or a jailbreak package source that can supply it).

No specific iPhone or iPad model beyond the validated iPad mini 2 is named here, because no other model has completed the project's real-device matrix. If you try one, please [report it](#contributing) — that is how coverage grows. Community testing is welcome.

#### Not supported

- **Stock / non-jailbroken iOS.** There is no way to install native binaries, LaunchDaemons, or system services without a jailbreak.
- **Devices that cannot execute the arm64 iOS binaries** provided by this project's releases.
- **Environments where the installer cannot obtain root privileges** (installation writes system paths and manages system services).
- **Configurations without the jailbreak facilities the installer depends on**, such as LaunchDaemon support or the ability to run native unsigned / ad-hoc-signed binaries.
- **Rootless layouts that do not expose the traditional paths** below, unless explicitly documented as validated in the future. The current installer expects the traditional filesystem layout (see [Rootful / rootless](#rootful--rootless)).

Newer chips (including arm64e devices) and newer iOS generations are **not claimed as supported** merely because the deployment target is iOS 12.0. They belong in the "expected compatible, currently unverified" group at best until real hardware results land.

### Jailbreak compatibility

#### Validated jailbreak environment

- **Amethyst + Procursus**
- **Semi-untethered** (the jailbreak becomes inactive after a full reboot and must be reactivated)
- **iOS 12.5.7**
- Tested on the **iPad mini 2** above, including a full reboot plus manual jailbreak reactivation with automatic Agent and Hub return.

#### Other jailbreaks

Other jailbreaks and bootstraps that provide the required root / bootstrap environment *may* work, but they are **currently unverified** until tested on real hardware.

Concretely, a different jailbreak / bootstrap has a reasonable chance of working only if it provides all of the following, which is what the installer (the `install.sh` bootstrap plus the `scripts/ios/install-beszel.sh` lifecycle engine) actually uses:

- Native arm64 command execution (ability to run the downloaded Agent / Hub binaries).
- Root or `sudo` access for the install session.
- Writable install locations at the validated paths (`/usr/local/bin`, `/var/lib`, `/Library/LaunchDaemons`, `/var/log`).
- `launchd` / LaunchDaemon management via `launchctl` (load, unload, list).
- `ldid` availability (or an `apt` source that can install it) for on-device pseudo-signing.
- `curl` plus network access to GitHub releases, and a SHA-256 tool for checksum verification.
- A writable jailbreak filesystem / bootstrap with standard Unix behavior.

No other named jailbreak (or bootstrap) is listed as supported here, because no repository evidence or real-device run backs such a claim yet. "Technically plausible" is not the same as "validated by this project."

### Rootful / rootless

"Rootful" here just means the classic jailbreak filesystem layout where traditional system paths such as `/usr/local/bin`, `/Library/LaunchDaemons`, and `/var/lib` are directly writable. "Rootless" jailbreaks remap or restrict those paths.

The current validated installer uses the **traditional system paths**:

- Binaries: `/usr/local/bin/beszel-agent`, `/usr/local/bin/beszel-hub`
- Service definitions: `/Library/LaunchDaemons/dev.beszel.agent.plist`, `/Library/LaunchDaemons/dev.beszel.hub.plist`
- Data: `/var/lib/beszel-agent`, `/var/lib/beszel-hub`
- Installer state: `/var/lib/beszel-ios/install-state`

If those directories are missing, the installer tries to create them; if they are not writable, it aborts with an explicit message that rootless layouts are not supported yet. So: the validated Procursus-based setup works, while **rootless jailbreak environments have not received equivalent validation** and should currently be treated as unsupported. The installer paths were deliberately not redesigned as part of this documentation update.

## Agent, Hub, or both?

Not sure which option to pick in the installer? Here's the simple version:

- **Agent** — "watch this device." It quietly measures this iPhone/iPad and sends the numbers to a Beszel Hub running somewhere else.
- **Hub** — "the dashboard." It collects numbers from Agents, stores history, and shows the web interface you log into.
- **Both** — "self-contained." The iOS device watches itself *and* hosts its own dashboard, so you can point a browser at the iPad itself.

Most people adding an old iPad to an existing setup just need the **Agent**. Pick **Both** if you want the iPad to work standalone.

Default ports: Agent `45876`, Hub `8090`. You can check the Hub locally at `http://127.0.0.1:8090/api/health`.

## Battery monitoring

Because this is iOS, the Agent also reports battery information — percentage and charging state — right next to CPU and memory in the dashboard. No extra setup needed.

## After reboot

> **Note:** the validated jailbreak is **semi-untethered**. A full device reboot temporarily disables the jailbreak environment — that's normal jailbreak behavior, not a Beszel problem.

What this means in practice:

- Beszel's files and data stay on the device across a reboot. A reboot does **not** delete Beszel.
- After rebooting, reactivate your jailbreak the way you normally do (for the validated setup, that means manually reactivating Amethyst).
- You do **not** need to reinstall Beszel.
- Once the jailbreak is active again, the Beszel Agent and Hub services return automatically.

This reboot-and-reactivation sequence was part of the real-device validation: after reboot plus manual jailbreak reactivation, both LaunchDaemons were back, the Hub answered healthy, the Agent reconnected with fresh statistics, and data and history were intact.

## Updating

Re-run the same installer command:

```sh
curl -fsSL https://raw.githubusercontent.com/nghianguyen150612/Beszel-iOS/iOS/install.sh | sudo sh
```

Choose **Update**. Your settings and Hub data are kept, the new binaries are verified before anything is swapped, and the previous version is kept as a backup with automatic rollback if the new one fails to start.

`beszel-ios update` refreshes the Beszel **application** binaries only. It never replaces the installed Beszel-iOS manager: to adopt a newer lifecycle engine, re-run the bootstrap command above, which fetches a bootstrap carrying the matching pinned engine checksum (see [Installer architecture](#installer-architecture)).

## Repair and reconfigure

The same installer menu also offers:

- **Diagnostics** — detailed read-only observations for Agent and Hub (installed files, service state, ports, health). Diagnostics never print the Agent key value.
- **Repair** — conservative fixes such as restarting a service in place, restoring a missing binary, or recreating a confirmed-broken configuration, with automatic rollback to the previous configuration if the fix fails.
- **Reconfigure** — change the Agent port/key or Hub port through a validated transaction that backs up the current configuration first.

The persistent CLI additionally provides the read-only `doctor` health assessment and safe start/stop/restart/status controls for the installed LaunchDaemons; see [CLI administration](#cli-administration).

## Uninstalling

Run the installer command again and choose **Uninstall**, then pick Agent, Hub, or both.

A normal uninstall removes the app and its services but **keeps your data**, so you can reinstall later without losing history. Wiping data is a separate, explicit step: the installer only deletes data when you type the exact confirmation phrase it shows you on screen. Anything else keeps your data safe.

Uninstall works offline and does not need to download or sign anything.

## Data locations

Useful if you make backups:

- Agent data: `/var/lib/beszel-agent`
- Hub data (database, accounts, history): `/var/lib/beszel-hub`
- Installer release state: `/var/lib/beszel-ios/install-state`

Never delete these by hand — use the installer menus. The Hub folder in particular holds your accounts and history.

## Beginner FAQ

**Do I need a jailbreak?**
Yes. Beszel-iOS installs native binaries and system services, which stock iOS does not allow.

**Can I install this on stock (non-jailbroken) iOS?**
No. Stock iOS is not supported.

**Can I install only the Agent?**
Yes. That is the most common setup: the iPad reports to a Hub running elsewhere.

**Can the iPad run the Hub too?**
Yes. The Hub runs natively on the validated iPad and serves the normal Beszel dashboard. You can also run Agent + Hub together on the same device so it monitors itself.

**Will Beszel disappear after I reboot?**
No. Files and data stay on the device. Because the validated jailbreak is semi-untethered, you reactivate the jailbreak after a reboot; the Agent and Hub then start again by themselves.

**Can I try another iPhone or iPad?**
You can try if it meets the criteria under [Likely / expected to be compatible](#likely--expected-to-be-compatible-not-yet-validated), but treat it as unverified and please report your result. Only the iPad mini 2 setup above is validated.

**Is this official Beszel software?**
No. This is an unofficial community port. Upstream Beszel does not publish or support iOS builds.

## Known limitations

- Real-device validation so far covers only the iPad mini 2 / A7 / iOS 12.5.7 / Amethyst + Procursus setup above. Other hardware and jailbreak combinations are unverified.
- A jailbroken device with root access is required. Stock iOS and rootless layouts are not supported by the current installer.
- After a full reboot, the semi-untethered jailbreak must be reactivated before Beszel's services can run again (they then return on their own).
- This is an unofficial community port, not an upstream-supported iOS release.

## Releases

Current iOS binary release: **v0.19.0-ios.1**

The version has two parts: `0.19.0` is the upstream Beszel version this port is based on, and `ios.1` is the iOS-port revision for that base. The installer itself is versioned separately (currently `1.0.0`).

Each release publishes three files: the Agent binary, the Hub binary, and a checksum file the installer verifies before installing anything. You can browse them under [GitHub Releases](https://github.com/nghianguyen150612/Beszel-iOS/releases).

## Documentation

For details beyond this page:

- [iOS port status](docs/ios-port-status.md) — what works and what's been tested
- [Real-device validation](docs/device-validation.md) — full results from the reference iPad
- [Port architecture](docs/architecture.md) — how the pieces fit together on iOS
- [Building iOS binaries](docs/ios-build-notes.md) — how releases are built
- [Upstream synchronization](docs/upstream-sync.md) — how the port stays close to upstream Beszel

### Translations

Vietnamese translation: [README.vi.md](README.vi.md)

## Building from source

Most users never need this — the installer already gives you ready-made binaries.

If you want to build iOS binaries yourself, you'll need macOS with the Xcode iPhoneOS SDK, plus Go and Bun, following the repository's build workflow. See [Building iOS binaries](docs/ios-build-notes.md) for the full steps.

## Contributing

The most helpful contribution right now is testing on more hardware. If you try Beszel-iOS on another jailbroken iPhone or iPad, please open an issue or discussion with:

- device model (e.g. iPad mini 2)
- `hw.machine` value (e.g. `iPad4,4`)
- iOS version
- jailbreak and bootstrap (e.g. Amethyst + Procursus)
- whether you ran Agent, Hub, or both
- what worked, and relevant logs if something failed

Build fixes, upstream-parity fixes, and documentation improvements are also welcome. Please target the `iOS` branch.

> Don't include private keys, passwords, tokens, or copies of your Hub database in reports.

## Upstream and credits

Beszel is by [henrygd](https://github.com/henrygd/beszel). This repository is an unofficial community iOS port maintained on the `iOS` branch — please direct general Beszel questions to the upstream project.

## License

MIT License — see [LICENSE](LICENSE). Original copyright retained:

> Copyright (c) 2024 henrygd
