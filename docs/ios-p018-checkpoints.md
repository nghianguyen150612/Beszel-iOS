# P018 checkpoint ledger

## Starting point

- Integration baseline: P017 Gate E normal merge `f9d6500c7a68572dc8e464dd305d79bf6c1e79a9`.
- Branch: `ios/p018-device-validation`, based exactly on that merge.
- No P018 hardware test performed as of the initial checkpoint.
- Production `iOS` and frozen `main` remain protected.
- Candidate source `5a7351f752376e206a3e1809b3698bb03cc87b98`; SHA evidence in `docs/ios-release-candidate-artifacts.json`.

## Checkpoint A — Read-only preflight and safe handoff

Implemented:
- `scripts/ios/p018-device-preflight.sh`: read-only local identity/path/hash/service inventory. Requires no network and makes no mutation. Redact output before attaching.
- `docs/ios-p018-device-handoff.md`: reference-device matrix, two-stage operator approval, backup requirements, staged candidate/rollback, jailbreak-reactivation steps and release stop conditions.
- `docs/ios-p018-validation.json`: structured Gate F matrix with all physical and signing checks explicitly PENDING.

Source scope: preflight and evidence only; no installation, signing, trust-cache changes, LaunchDaemon changes, reboot, database modifications or release.

## Initial next required step (superseded by Checkpoint B)

Get explicit confirmation that the **physical reference iPad** and recovery path are available. Inspect the preflight script, transfer it through an authorized channel, run `sh /var/tmp/p018-device-preflight.sh` on the actual device, and provide redacted results. Do not proceed to candidate staging until complete cold backup, restore rehearsal and operator approval.

P018 cannot reach PASS without real hardware evidence. Native Wi-Fi RSSI remains DEFERRED without an approved ABI/authorization experiment. The high-severity Go multipart and build-only braces advisories remain OPEN.

**Initial Gate F snapshot: PENDING — REFERENCE DEVICE UNAVAILABLE.**

## Checkpoint B — Host-side preflight safety and repository gates

Recorded at `2026-10-18T05:09:23Z` in ChatGPT Work. GitHub refs were fetched and reconciled before continuing: PR #22 remains Draft; its head and `origin/ios/p018-device-validation` are `25aa2589182d1460839a300849f71f394472a4fb`; the integration base is still `f9d6500c7a68572dc8e464dd305d79bf6c1e79a9`; `iOS` and `main` remain at `0c3e4cd22ac24264834c401ad412e44aeda4e4f3` and `bb23f99b3eeaced9608e11e280e7c986dba650be`.

No authorized terminal/SSH session or target endpoint for the iPad was available in this ChatGPT Work execution. No network scan or device connection attempt was made. Therefore no hardware evidence, backup, candidate signing/staging, native launch, service change, or reboot occurred. Candidate binaries were not executed by any host check below; candidate hashes remain the P017 values recorded above.

Added `tests/ios-p018-device-preflight-test.sh` and wired it into the installer lifecycle workflow for P018 script, test, and checkpoint changes. The host harness runs the preflight with guarded command stubs and proves the script uses only `launchctl list`, calls no guarded mutator/network/database command, and rejects unexpected arguments before service inspection. This is a tooling safety test only; it does not emulate an iPad.

Host check results, all recorded at the result checkpoint above:

| Test ID | Command | Actual result | Gate meaning |
| --- | --- | --- | --- |
| `P018-CLOUD-01` | `sh -n scripts/ios/p018-device-preflight.sh && sh -n tests/ios-p018-device-preflight-test.sh` | Exit 0 | Host syntax only |
| `P018-CLOUD-02` | `sh tests/ios-p018-device-preflight-test.sh` | Passed | Preflight safety harness only |
| `P018-CLOUD-03` | `sh tests/bootstrap-test.sh` | 54 passed, 0 failed | Bootstrap fixtures |
| `P018-CLOUD-04` | `sh tests/install-sh-test.sh` | 873 passed, 0 failed | Installer and read-only CLI fixtures |
| `P018-CLOUD-05` | `sh tests/upstream-sync-test.sh` | 10 passed, 0 failed | Sync fixtures |
| `P018-CLOUD-06` | `sh tests/ios-patch-sentinels.sh` | 45 passed, 0 failed | Source sentinels |
| `P018-CLOUD-07` | `python3 scripts/audit/validate-upstream-v021-matrix.py` | 178/178 upstream commits, 35/35 historical iOS commits, 0 unresolved rows | Accounting only |
| `P018-CLOUD-08` | `python3 scripts/audit/validate-upstream-v021-readiness.py` | 178 commits, 13 logical iOS patches, 21 P0 risks, 24 acceptance cases, 33 compatibility scenarios | Accounting only |
| `P018-CLOUD-09` | `git diff --check` | Exit 0 | Local whitespace check |

For each check, candidate binary hash is **N/A** because no Agent/Hub executable was run; none of these host results passes a physical acceptance case. GitHub CI for the changed workflow and preflight test must still complete after push. Gate F remains `BLOCKED — REFERENCE DEVICE UNAVAILABLE`.

## Next required step

Provide an authorized connection to the actual reference iPad and its safe recovery operator. Transfer the reviewed preflight through that authorized channel, run `sh /var/tmp/p018-device-preflight.sh`, and return sanitized output. Do not stop services, create a cold backup, sign or stage candidates, change live files, or reboot until the separate recovery and approval gates are met.
