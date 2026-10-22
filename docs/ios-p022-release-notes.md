# Beszel-iOS `v0.21.0-ios.1` candidate notes (draft)

**Not approved for production. Not published.** The proposed tag follows the repository's existing Beszel-iOS tag format. This candidate integrates the pinned upstream snapshot `b5357343550c8e2493d0e65472a06c1007396384` with the preserved native iOS patches. It does not claim that upstream published a newer stable release.

## Candidate scope

- Integrates 41 post-v0.21 upstream-main commits on the pinned snapshot, with 178 frozen historical upstream audit rows, 35 historical iOS commits, and 13 logical iOS patchsets accounted for.
- Provides native iOS ARM64 Agent and Hub builds, CPU telemetry, network monitoring, and TLS alert updates.
- Includes OAuth and authorization changes, frontend and localization updates, and upstream database migrations.
- Uses Go 1.27.2 with the reviewed Apple A7 runtime guard and official x509 security backport; candidate builds target iOS 12.0 or later.
- Retains the rootful `beszel-ios` installer, per-component Agent/Hub state, LaunchDaemon lifecycle, checksum verification, staged updates, and backup-based rollback.

## Security and validation

The candidate is subject to exact-source Go and JavaScript advisory scans, Go and frontend regression suites, genuine synthetic historical Hub migration and recovery, installer lifecycle checks, and hosted macOS/Xcode native Agent/Hub validation. The proposed notes do not represent those P022 checks as complete until the exact PR head evidence is recorded.

Open or pending advisory dispositions include GO-2026-5932 and GHSA-vfj7-8cjw-p6xm; GO-2026-6809 may remain visible as raw scanner occurrences while the official x509 backport and tests pass. GO-2026-6608 must be checked again against the exact candidate toolchain.

## Limitations

- No physical iPad compatibility or runtime test is claimed. P018 Gate F remains PENDING.
- No public release, production rollout, or signing identity is approved by P022.
- iOS Wi-Fi diagnostics remain deferred; this candidate does not claim private MobileWiFi support.
- A migrated PocketBase database must not be downgraded by replacing only the Hub binary; use a verified cold backup and approved full restore.
- Checksum verification and `ldid` signing do not establish Apple's code-signing trust.
