# P026 device preflight

**Status: PENDING operator output.** No physical-device observation has been made in this workspace. Gate F remains PENDING.

## Target contract

- iPad mini 2 A1489, expected hw.machine iPad4,4
- Apple A7, ARM64
- iOS 12.5.7
- Amethyst / Procursus rootful jailbreak, semi-untethered

Stop if the observed model, OS, architecture, or filesystem layout differs. Do not substitute hosted CI or historical v0.19 results for physical evidence.

## Reviewed read-only helper

P018 PR #22 remains Draft and contains the existing preflight helper. It is read-only by inspection: it queries uname, sysctl, plutil, id, command/path presence, file hashes, launchctl list, and df. It contains no installer, service-control, network-download, process-stop, database-writer, or reboot operation. Its pinned source is:

- P018 head commit: 8e951a005dbd684f1b3e09c35a27ed1d08001d14
- Path: scripts/ios/p018-device-preflight.sh
- SHA-256: 8044ccfd6ccd199c01746bcc67985fabf3c5c1a9
- Script blob SHA: 8044ccfd6ccd199c01746bcc67985fabf3c5c1a9

Extract the exact reviewed file on the workstation and verify it before using it:

~~~sh
git fetch origin ios/p018-device-validation
git show 8e951a005dbd684f1b3e09c35a27ed1d08001d14:scripts/ios/p018-device-preflight.sh > /tmp/p018-device-preflight.sh
echo '8044ccfd6ccd199c01746bcc67985fabf3c5c1a9  /tmp/p018-device-preflight.sh' | shasum -a 256 -c
~~~

Run it through an already configured SSH alias without leaving a helper file on the iPad:

~~~sh
ssh <configured-alias> 'sh -s' < /tmp/p018-device-preflight.sh
~~~

The SSH command is operator-run. This workflow has not attempted a connection, scan, or login to the iPad.

## Evidence to return

Paste the script output with usernames, hostnames, IP addresses, process IDs, and private values removed. Keep the hardware model, OS version, architecture/cputype, effective uid, availability of launchctl/ldid/checksum tools, rootful path states, installed Beszel binary hashes/presence, Agent/Hub launchd labels and states, and /var capacity.

The helper does not establish recovery readiness. Along with the output, report whether physical access and stable power are available, and describe the known manual Amethyst reactivation and recovery-terminal/SSH path after an ordinary full reboot. Do not reboot to test the path.

If the existing read-only manager command is available, its status output may supplement the script. Redact PIDs and private values. Do not run install, update, repair, service start/stop/restart, configuration changes, database commands, or reboot in Phase A.

## Current result

No device output has been supplied yet. Preflight is PENDING; identity, rootful layout, current installation, service status, storage, and recovery capability are unverified.
