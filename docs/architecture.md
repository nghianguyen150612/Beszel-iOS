# Architecture

## Upstream relationship

```text
henrygd/beszel
      |
      v
    main            upstream-aligned branch
      |
      + iOS compatibility work (8 files: battery, system metadata,
      |                         runtime patch, 2 workflows)
      v
     ios            active iOS port branch
```

Design rule: `ios` stays recognizable as Beszel. No Agent rewrite, no Hub rewrite, no database-semantics change, no removed upstream features.

## Runtime relationship (unchanged from upstream)

```text
Beszel Hub
    |
    | Beszel Agent protocol (same as upstream)
    v
Beszel Agent on iOS
```

The Agent collects host metrics and serves them to the Hub on its configured port; the Hub stores history and serves the dashboard. iOS only changes *how* the Agent obtains battery/system metadata and *how* both binaries are built — not the protocol. Protocol details live in the upstream source and are not restated here.

## On-device runtime layout (iOS)

```text
/usr/local/bin/beszel-agent          Agent binary (ldid-signed)
/usr/local/bin/beszel-hub            Hub binary (ldid-signed)

/var/lib/beszel-agent                Agent state
/var/lib/beszel-hub                  Hub DB / accounts / config (NEVER disposable)

/Library/LaunchDaemons/dev.beszel.agent.plist
/Library/LaunchDaemons/dev.beszel.hub.plist
```

Default ports: Agent `45876`, Hub `8090`. Hub health: `http://127.0.0.1:8090/api/health`.

Binaries must live under `/usr/local/bin` (`chown root:wheel`, `chmod 755`, `ldid -S`). Home-directory or `/tmp` execution has previously hit iOS sandbox problems.

## iOS-specific code map

- `agent/battery/battery_ios.go` — ioreg charger-class battery reader.
- `agent/battery/battery_darwin.go` — macOS path, now `darwin && !ios`.
- `agent/system.go` — skips Darwin `cpu.Info()` on iOS, calls `adjustPlatformSystemDetails()`.
- `agent/system_platform_ios.go` / `agent/system_platform_other.go` — iOS sysctl/plist metadata vs. no-op.
- `.github/scripts/patch-go-ios-arm64-runtime.py` — A7 `procyield` workaround (build-time, macOS runner).
- `.github/workflows/ios-agent-probe.yml` / `ios-hub-probe.yml` — device-targeted builds.

## Future distribution architecture (planned, not implemented)

```text
GitHub Release
     |
     +-- beszel-agent-ios-arm64
     |
     +-- beszel-hub-ios-arm64
     |
     +-- SHA256SUMS
```

```text
install.sh
     |
     +-- Install Agent
     +-- Install Hub
     +-- Install Both
     +-- Update (preserve /var/lib/beszel-hub)
     +-- Repair / reconfigure
     +-- Uninstall (explicit confirmation for data)
```

A future `packaging/launchd/` directory will hold the two LaunchDaemon plists; `scripts/` will hold device-side helpers. Neither exists yet — see [ios-port-status.md](ios-port-status.md) and [ios-build-notes.md](ios-build-notes.md).
