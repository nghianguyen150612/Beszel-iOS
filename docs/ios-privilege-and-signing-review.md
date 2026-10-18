# P016 privilege and signing review

## Agent

The Agent remains a rootful LaunchDaemon because its existing iOS collectors query system-level device and network information, expose the configured Agent listener, manage its fingerprint/configuration, and write its own service logs and state. The Hub authenticates to the Agent over the existing WebSocket or SSH transport. P016 adds a private, short-lived connection marker under the Agent's temporary directory; it is created only after an authenticated transport is active and removed on disconnect. The marker is root-owned, mode `0600`, checked without following symlinks, and contains no credential.

The new `beszel-agent connection` command checks that marker. The existing `beszel-agent health` command continues to check process/event-loop liveness; it is not used as proof of Hub connectivity.

## Hub

The Hub remains a rootful LaunchDaemon to preserve the established install and PocketBase data layout. It needs write access to `/var/lib/beszel-hub` for PocketBase databases and migrations, Hub SSH identity and configuration, plus its configured listener and logs. Its state and backup operations remain scoped to installer-owned paths. P016 does not recursively change ownership of user data.

## Installer

Root is required for binaries and LaunchDaemon files under `/usr/local/bin` and `/Library/LaunchDaemons`, state/manager and transaction backups under `/var/lib/beszel-ios`, ownership/mode correction of individual managed files, `ldid -S`, and iOS 12 compatible `launchctl` operations. Read-only status, diagnostics, and doctor do not require root. Mutations share one private atomic lock.

The backup stores database and key bytes because complete rollback requires them. The archive and transaction directory are private, hashes and metadata are verified before restore, and only fixed allow-listed paths are replaced. The installer does not export backup contents or print keys.

## Signing and entitlements

P016 retains existing ad-hoc `ldid -S` signing. No entitlement, trust-cache, jailbreak signing, MobileWiFi, or private Wi-Fi change is made. The synthetic tests replace `ldid` with a mock and validate only that the operation is sequenced and failures roll back. The iOS Mach-O parser runs before install and after staging, but cannot establish a trusted signature.

P018 must validate the minimum permissions and actual signing behavior on the reference iPad, including LaunchDaemon execution, Agent system metrics, Hub database/key access, Agent-to-Hub connection state, jailbreak reactivation, and reboot persistence. Until that evidence exists, no physical-device privilege or signing claim is made.
