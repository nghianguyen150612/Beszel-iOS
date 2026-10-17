# P016 installer security review

## Scope

The rootful installer manages only the existing iOS paths defined by the lifecycle engine: `/usr/local/bin`, `/var/lib/beszel-ios`, `/var/lib/beszel-agent`, `/var/lib/beszel-hub`, `/Library/LaunchDaemons`, and the configured Beszel log directory. It rejects rootless layouts. This review covers installer and service behavior; it does not claim that a cloud fixture proves iOS signing trust or on-device behavior.

## Bootstrap and artifact integrity

`install.sh` fetches the lifecycle engine only from the canonical HTTPS URL on the `iOS` branch, stores it in a private temporary directory, checks the 64-character SHA-256 pin, then executes that local copy. The engine and matching pin are changed together. A download, redirect, checksum, or local write failure stops before the downloaded file executes. The installed manager is persisted from the verified local engine.

This pin gives a fetched bootstrap a byte-for-byte engine check. It is not an independently signed bootstrap: someone able to replace the bootstrap and its pin at the source can replace both. Release assets are selected from one resolved immutable tag and checked against that release's `SHA256SUMS`. Those checks detect corrupt or mixed payloads, but the manifest is not an independent release signature.

Before install or update, the engine checks that each file is a thin little-endian ARM64 baseline executable Mach-O, has an iOS device platform load command, and targets iOS 12.0 or earlier. It rechecks the staged copy after `ldid`. This is a format and target check; it does not prove that `ldid` created a trusted signature or that iOS will launch the binary. The P015 build artifacts used as positive fixtures are from hosted run [39014286778](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39014286778), source `b83e179bd84b2177cd7fc8b5b0f5460d5f2411ba`, artifact `12093514420`. Both artifact checksums match, and hosted validation records Go 1.27.1, `GOOS=ios`, `GOARCH=arm64`, `CGO_ENABLED=1`, iOS platform 2, minimum iOS 12.0, and clean VCS metadata.

## Rootful paths and filesystem controls

Mutation requires root and the standard rootful layout. Before use, the engine checks parent components and managed directories for symlinks, non-directory replacements, group/world write permission, and unexpected ownership. Installer data snapshots reject symlinks, special files, unexpected hardlinks, unsafe modes, and paths outside the fixed allow-list. Temporary staging and transaction backup directories are private. The atomic lock serializes lifecycle mutations; a malformed or symlinked lock fails closed. Cleanup removes only a work directory created by that invocation.

Install state is written through a same-directory temporary file and rename. The state contains release tags and SHA-256 values, not SSH keys or credentials. Ordinary uninstall preserves Agent and Hub data. Purge uses the existing exact confirmation contract independently for each component; blank input, cancellation, and errors do not authorize deletion.

No operation recursively changes ownership of user databases or Hub key material. The transaction archive records and verifies owner, group, mode, type, size, and file hash, then restores only allow-listed paths.

## Read-only commands and network exposure

`status`, `diagnostics`, and `doctor` take no mutation lock and do not perform downloads, package operations, database changes, service changes, or install-state writes. Agent public-key state is reported as configured/absent; the key itself is not printed. Malformed release-state values are not echoed. A pending or malformed Hub transaction is reported without recovery; doctor fails when recovery is pending.

Hub and Agent continue to use the established rootful LaunchDaemons and configured listening ports. Existing Hub request validation is unchanged. The production Hub TLS server continues to disable HTTP/2 negotiation, and the 16-range limit remains in place. No new dependency or major toolchain upgrade is included.

## Signing and residual risks

The installer invokes the existing `ldid -S` operation for staged binaries. Cloud tests mock it and make no claim about a trusted iOS signature. P016 adds no entitlements, jailbreak trust-cache changes, private Wi-Fi access, or MobileWiFi linkage. Runtime privilege, real signing, LaunchDaemon persistence after jailbreak reactivation, and reboot behavior remain P018 physical-device work.

The documented high residuals remain open: reachable Go multipart advisory `GO-2026-6608` on the pinned Go 1.27.1 toolchain and build-only `GHSA-vfj7-8cjw-p6xm` for `braces`. The existing HTTP/2 and Range mitigations remain unchanged. P016 does not remove these findings or pass Gate E; P017 still owns release-candidate validation.
