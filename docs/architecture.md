# Architecture

## Upstream relationship

```text
henrygd/beszel
      |
      v
    main            upstream-aligned branch
      |
      + iOS compatibility work (battery, system metadata,
      |                         runtime patch, consolidated
      |                         build/release workflow)
      v
     ios            active iOS port branch (default branch)
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
- `.github/workflows/ios-build.yml` — consolidated pipeline (Agent + Hub + SHA256SUMS → one artifact; on `v*-ios.*` tags, a dependent job validates the tag and publishes the same three files as a Latest GitHub Release).

## Distribution status

**Existing:** consolidated CI pipeline. Each run produces `build/ios/` with exactly `beszel-agent-ios-arm64`, `beszel-hub-ios-arm64`, `SHA256SUMS`, uploaded as the `beszel-ios-arm64` artifact. Pushing a valid `v<upstream-version>-ios.<revision>` tag (matching `beszel.Version` in `beszel.go`, on `ios` history) publishes those exact three files as a non-draft, non-prerelease GitHub Release marked Latest. Upstream tag-triggered automation (`release.yml`, `docker-images.yml`) ignores `v*-ios.*` tags so iOS releases stay clean. The `install.sh` one-line installer (v0.4.0) consumes the Latest release for fresh Agent / Hub / Agent+Hub installs (checksum verification, `ldid` signing, on-device LaunchDaemon generation, Hub health check) and for transactional updates: it resolves the Latest tag once per run, pins all downloads to that immutable tag, tracks installed releases in `/var/lib/beszel-ios/install-state` (installed binaries are `ldid`-signed, so their hashes are never compared against `SHA256SUMS`), stages signed binaries before downtime, keeps `/usr/local/bin/*.bak` backups, rolls back automatically on failed health/startup checks, preserves plists byte-for-byte, and never touches `/var/lib/beszel-hub`. It also offers read-only diagnostics (the Agent key value is never displayed), conservative repair, and reconfiguration through validated plist transactions backed up to `/Library/LaunchDaemons/*.plist.bak` with automatic config rollback; reconfiguration and restart-only repair never alter install state, and only repairs that install a freshly downloaded Latest binary record the new release. Application uninstall (Agent, Hub, or Agent+Hub) is transactional with rollback, needs no network or `ldid`, always preserves `/var/lib/beszel-agent` and `/var/lib/beszel-hub` by default, and offers data purge only on exact typed confirmation (`DELETE AGENT DATA` / `DELETE HUB DATA`); per-component state is cleared and empty installer metadata is removed with exact paths only.

No further installer lifecycle work is planned: install, update, repair/reconfigure, and uninstall are all implemented. Remaining work is validation (real-device end-to-end runs, additional iOS devices/SoCs).

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
     +-- Update (preserve /var/lib/beszel-hub)      [implemented]
     +-- Repair / reconfigure                       [implemented]
     +-- Uninstall (data preserved; purge needs typed confirmation) [implemented]
```

A future `packaging/launchd/` directory will hold the two LaunchDaemon plists; `scripts/` will hold device-side helpers. Neither exists yet — see [ios-port-status.md](ios-port-status.md) and [ios-build-notes.md](ios-build-notes.md).
