# Beszel-iOS service and CLI lifecycle

## CLI surface

The persistent command is `/usr/local/bin/beszel-ios`; its lifecycle engine is `/var/lib/beszel-ios/manager.sh`.

| Command | Access | Effect |
| --- | --- | --- |
| `beszel-ios`, `menu`, `help`, `version` | read-only | Show menu or local CLI metadata. |
| `status [agent|hub|both]` | read-only | Inspect installation and LaunchDaemon state. |
| `diagnostics [agent|hub|both]` | read-only | Report local dependencies, ports, health, and redacted key state. |
| `doctor [agent|hub|both]` | read-only | Validate managed files, platform, dependencies, service state, and Hub health. |
| `install agent|hub|both` | mutation | Install requested components in the existing rootful layout. |
| `update agent|hub|both` | mutation | Update binaries; Hub updates use a complete data/service transaction. |
| `repair agent|hub|both` | mutation | Repair damaged managed files while preserving data. |
| `reconfigure agent|hub` | mutation | Replace service configuration after explicit confirmation. |
| `uninstall agent|hub|both [--purge]` | mutation | Remove binaries/plists; preserve data unless separately confirmed for purge. |
| `service agent|hub|both start|stop|restart|status` | mutation except `status` | Control existing LaunchDaemons using the iOS 12 `launchctl` interface. |

Read-only commands do not acquire the mutation lock or invoke network downloads, package managers, database writers, state writers, or service controls. A pending Hub transaction is reported. Doctor reports failure but leaves recovery untouched. A mutating command acquires the atomic lock, verifies any pending transaction, recovers it from a verified snapshot, and only then continues.

## Service ordering

| Operation | Order |
| --- | --- |
| Start both | Hub, then Agent. |
| Stop both | Agent, then Hub. |
| Restart both | Stop Agent, stop Hub, start Hub, start Agent. |
| Hub transaction recovery | Stop Agent, stop Hub, restore and verify complete snapshot, start prior Hub state, then prior Agent state. |

`svc_stop` uses `launchctl stop <label>` and leaves a previously loaded but idle service loaded. Start/enable uses `launchctl load -w <plist>`; unload uses the existing legacy interface. The engine does not assume macOS `bootstrap` or `bootout` commands exist on iOS 12. Existing labels remain `dev.beszel.hub` and `dev.beszel.agent`.

After a Hub candidate starts, the installer polls the configured loopback `/api/health`. If an Agent was installed and running before the transaction, its new `connection` subcommand must prove a recent authenticated WebSocket or SSH connection to the Hub. The separate `health` subcommand only proves process/event-loop liveness. If Hub health, Agent connectivity, state commit, rollback, or service-state restoration fails, the transaction reports failure; uncertain recovery leaves both services stopped and the journal available for retry.

## Validation and hardware boundary

The installer test suite covers parser grammar, data-preserving uninstall and explicit purge, order and error propagation, read-only command behavior, secret redaction, missing/stale services, unhealthy Hub, Agent disconnect, rollback, and retry. These are disposable filesystem and mocked-command tests. P018 remains responsible for LaunchDaemon behavior after device reboot and jailbreak reactivation.
