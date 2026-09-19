# Beszel iOS

Run Beszel Agent and Hub natively on a jailbroken iPhone or iPad — turn old iOS hardware into a lightweight monitoring node or self-hosted server.

[![License: MIT](https://img.shields.io/github/license/nghianguyen150612/beszel-ios)](LICENSE)
![Beszel base 0.19.0](https://img.shields.io/badge/Beszel%20base-0.19.0-blue)
![Tested on iPad mini 2 · iOS 12.5.7](https://img.shields.io/badge/tested-iPad%20mini%202%20%C2%B7%20iOS%2012.5.7-green)

[Beszel](https://github.com/henrygd/beszel) is a lightweight, self-hosted server monitoring platform. You install a small **Agent** on each machine you want to watch, and a **Hub** collects the numbers and shows them in a web dashboard — CPU, memory, disk, network, and more. This project is an **unofficial community port** that runs both the Agent and the Hub directly on jailbroken iOS, with no Linux VM or container needed.

> Upstream project: **Beszel by henrygd** — <https://github.com/henrygd/beszel>
>
> This is a community port. It is not affiliated with or endorsed by the upstream project.

## Why Beszel iOS?

Have an old iPhone or iPad sitting in a drawer? If it's jailbroken, it can still be useful.

Beszel iOS lets you reuse that device as a small always-on machine: run the monitoring Agent on it, check it from your normal Beszel dashboard, and — if you want — host the dashboard itself on the same iPhone or iPad. Everything runs as native arm64 iOS binaries and starts automatically with the system once your jailbreak environment is active.

In short: old iPad in, live system graphs out.

## Features

- **Native Beszel Agent for iOS** — reports CPU, memory, disk, network, load, and system info to any Beszel Hub.
- **Native Beszel Hub for iOS** — serves the familiar Beszel web dashboard and stores history right on the device.
- **Battery telemetry** — shows battery percentage and charging state alongside the usual server metrics.
- **Starts automatically** — Agent and Hub run as system services and come back after jailbreak reactivation.
- **Hub + Agent on one device** — one iPhone or iPad can monitor itself and show its own dashboard.
- **One-command installer** — install, update, repair, reconfigure, and uninstall from a simple menu. Downloads are checksum-verified, updates keep a backup with automatic rollback, and your data is preserved.
- **Safe by default** — normal uninstall removes the app but keeps your data; wiping data always asks for explicit confirmation.
- **Stays close to upstream** — same Agent + Hub design and protocol as regular Beszel, based on upstream 0.19.0.

## Quick Start

> **Before you start:** make sure your device meets the [Prerequisites](#prerequisites) below. In particular, you need a jailbroken device with SSH or terminal access.

SSH into your jailbroken iPhone or iPad, then run:

```sh
curl -fsSL https://raw.githubusercontent.com/nghianguyen150612/beszel-ios/ios/install.sh | sudo sh
```

That's it — one command, no cloning or extra downloads. Then:

1. Choose **Agent**, **Hub**, or **Agent + Hub** from the menu.
2. Follow the prompts (for the Agent you'll need your Hub's public key; for the Hub you'll pick a port).
3. If you installed a Hub, open its web address to create your account. If you installed an Agent, add it to your Hub as usual.

To do anything later — update, check status, fix, reconfigure, or uninstall — just run the same command again and pick the option you want.

## Prerequisites

- A jailbroken **arm64** iPhone, iPad, or iPod touch.
- A working jailbreak environment with standard Unix tools (`curl`, `launchctl`), network access, and a SHA-256 tool (`sha256sum`, `shasum`, or `openssl`).
- A terminal on the device, or SSH access to it.
- The ability to run commands as root (`sudo`) for installation.

The validated setup is an **Amethyst + Procursus** environment (see [Compatibility](#compatibility)). Other jailbreak setups may work, but they haven't received the same real-device testing yet. The installer will offer to install `ldid` (used to sign iOS binaries) if it's missing.

## Compatibility

| Device | Chip | iOS | Jailbreak | Status |
| --- | --- | --- | --- | --- |
| iPad mini 2 (iPad4,4 / A1489) | Apple A7 | 12.5.7 | Amethyst + Procursus (semi-untethered) | **Tested** |

Other jailbroken arm64 iPhones and iPads may work, but they are currently community-tested / unverified. Stock (non-jailbroken) iOS is not supported.

If you try another device, please [report it](#contributing) — that's how the table grows.

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

- Beszel's files and data stay on the device across a reboot.
- After rebooting, reactivate your jailbreak the way you normally do.
- Once the jailbreak is active again, the Beszel services return automatically. You don't need to reinstall anything.

## Updating

Re-run the same installer command:

```sh
curl -fsSL https://raw.githubusercontent.com/nghianguyen150612/beszel-ios/ios/install.sh | sudo sh
```

Choose **Update**. Your settings and Hub data are kept, the new binaries are verified before anything is swapped, and the previous version is kept as a backup with automatic rollback if the new one fails to start.

## Uninstalling

Run the installer command again and choose **Uninstall**, then pick Agent, Hub, or both.

A normal uninstall removes the app and its services but **keeps your data**, so you can reinstall later without losing history. Wiping data is a separate, explicit step: the installer only deletes data if you type the exact confirmation phrase (`DELETE AGENT DATA` or `DELETE HUB DATA`). Anything else keeps your data safe.

## Data locations

Useful if you make backups:

- Agent data: `/var/lib/beszel-agent`
- Hub data (database, accounts, history): `/var/lib/beszel-hub`

Never delete these by hand — use the installer menus. The Hub folder in particular holds your accounts and history.

## Known limitations

- Real-device testing so far centers on the iPad mini 2 / A7 / iOS 12.5.7 setup above. Other hardware and jailbreak combinations are not guaranteed yet.
- A jailbroken device is required. Stock iOS is not supported.
- After a full reboot, the semi-untethered jailbreak must be reactivated before Beszel's services can run again (they then return on their own).
- This is an unofficial community port, not an upstream-supported iOS release.

## Releases

Current iOS binary release: **v0.19.0-ios.1**

The version has two parts: `0.19.0` is the upstream Beszel version this port is based on, and `ios.1` is the iOS-port revision for that base. The installer itself is versioned separately (currently `1.0.0`).

Each release publishes three files: the Agent binary, the Hub binary, and a checksum file the installer verifies before installing anything. You can browse them under [GitHub Releases](https://github.com/nghianguyen150612/beszel-ios/releases).

## Documentation

For details beyond this page:

- [iOS port status](docs/ios-port-status.md) — what works and what's been tested
- [Real-device validation](docs/device-validation.md) — full results from the reference iPad
- [Port architecture](docs/architecture.md) — how the pieces fit together on iOS
- [Building iOS binaries](docs/ios-build-notes.md) — how releases are built
- [Upstream synchronization](docs/upstream-sync.md) — how the port stays close to upstream Beszel

## Building from source

Most users never need this — the installer already gives you ready-made binaries.

If you want to build iOS binaries yourself, you'll need macOS with the Xcode iPhoneOS SDK, plus Go and Bun, following the repository's build workflow. See [Building iOS binaries](docs/ios-build-notes.md) for the full steps.

## Contributing

The most helpful contribution right now is testing on more hardware. If you try Beszel iOS on another jailbroken iPhone or iPad, please open an issue or discussion with:

- device model (e.g. iPad mini 2)
- `hw.machine` value (e.g. `iPad4,4`)
- iOS version
- jailbreak and bootstrap (e.g. Amethyst + Procursus)
- whether you ran Agent, Hub, or both
- what worked, and relevant logs if something failed

Build fixes, upstream-parity fixes, and documentation improvements are also welcome. Please target the `ios` branch.

> Don't include private keys, passwords, tokens, or copies of your Hub database in reports.

## Upstream and credits

Beszel is by [henrygd](https://github.com/henrygd/beszel). This repository is an unofficial community iOS port maintained on the `ios` branch — please direct general Beszel questions to the upstream project.

## License

MIT License — see [LICENSE](LICENSE). Original copyright retained:

> Copyright (c) 2024 henrygd
