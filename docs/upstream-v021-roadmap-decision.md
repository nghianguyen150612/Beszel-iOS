# P003 roadmap reconciliation decision

**Decision:** The originally approved P004–P018 roadmap is authoritative. P002's
`integration_prompts` and `responsible_prompt` values remain unchanged audit
evidence. Where those fields route implementation work through the provisional
P007–P017 plan, the new
[`upstream-v021-implementation-crosswalk.csv`](upstream-v021-implementation-crosswalk.csv)
records the original label and the canonical owner side by side.

P004–P006 own the upstream history batches only: 25 commits through v0.20.0,
116 additional commits through v0.21.0, and the remaining 37 commits through
the frozen `bb23f99b3` target. Each upstream SHA also has one primary functional
owner under the approved roadmap. The crosswalk records both owners so history
integration does not replace implementation ownership.

P003 owns the readiness package and its structural checks. It does not become a
catch-all implementation owner. A commit that touches several areas receives a
primary functional owner based on its changed subsystem and P002 evidence paths;
the reason column records this choice. Related test, build, or release prompts
remain visible in the retained P002 row and the crosswalk's integration owner.

| Prompt | Canonical responsibility |
| --- | --- |
| P004 | Merge through v0.20.0 |
| P005 | Merge through v0.21.0 |
| P006 | Merge remaining frozen upstream commits |
| P007 | iOS build tags, CGO, SDK, A7 runtime |
| P008 | CPU/RAM/disk/battery/system metadata |
| P009 | WebSocket, SSH, TLS, headers, reconnect |
| P010 | Network monitors, SMART, ZFS, GPU, unsupported-platform behavior |
| P011 | iOS jailbreak package-update adapter |
| P012 | iOS Wi-Fi RSSI adapter/fallback |
| P013 | Hub security, PocketBase, migrations, authorization |
| P014 | Dashboard, alerts, monitors, localization |
| P015 | Dependencies, Bun, frontend embed, Hub build |
| P016 | Installer, CLI, services, releases |
| P017 | Tests, CI, regression and release candidate |
| P018 | Physical-device validation and release authorization |

The crosswalk has one row for each of the 178 canonical upstream SHA values and
one row for each of the 13 distinct logical iOS patches. Repeated historical
commits for a logical patch are intentionally collapsed to one patch row; the
P002 patch map remains the commit-level audit record. No P002 row is rewritten.

## Ownership rules used by the crosswalk

Each upstream row has one primary functional owner based on the commit's main
behavior and inspected source paths. The history batch owner remains a separate
field. Cross-cutting test files do not replace the owner of the behavior they
exercise; test-only commits map to P017. The primary-owner routing is:

* P007 owns iOS target/build configuration, CGO/SDK requirements, and the A7
  runtime patch.
* P008 owns Agent CPU/RAM/disk/battery/system telemetry and raw interface
  counters.
* P009 owns Agent/Hub transport, WebSocket/SSH lifecycle, TLS, headers, framing,
  serialization, reconnect, and cancellation.
* P010 owns monitor probing/schemas/history plus SMART, ZFS, GPU, and local
  collectors that are unsupported on iOS.
* P011 owns package-update collection and the future read-only jailbreak
  package-manager adapter; P012 owns Wi-Fi RSSI collection and its iOS
  adapter/fallback.
* P013 owns Hub security, account/record authorization, PocketBase configuration,
  migrations, and data ownership. P014 owns user-facing dashboard, alert,
  monitor, settings, and localization behavior.
* P015 owns dependencies, Bun, frontend embedding, frontend build behavior, and
  Hub artifact construction. P016 owns installers, CLI/service lifecycle, and
  release/image publishing. P017 owns test-only changes, CI verification,
  regression evidence, and documentation needed to audit/review the release
  candidate.

For a multi-area change, the crosswalk names the prompt responsible for its
primary behavior; the affected-subsystem and evidence columns retain the other
areas. This keeps, for example, monitor backend ownership (P010), monitor UI
(P014), transport reconnect (P009), and migration/authorization controls (P013)
distinct while preserving P002's original routing labels.

This is a routing decision for P003 onward. It does not begin an upstream
merge, change production behavior, approve a database upgrade, or authorize a
release.
