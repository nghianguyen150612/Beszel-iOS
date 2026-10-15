# P012A native Wi-Fi blocker resolution

P012 remains **BLOCKED**; Gate C remains **PENDING**. PR #14 remains Draft.
Starting feature: `88dcf9b89389cd58de6f4859d4619485451dfa82`; integration:
`33840093bb465f0b164bb7d66aebc8cd85f86a4e`. This continuation preserves original
P012 evidence. Normal iOS polling still returns nil; no native acquisition flag
exists. Linux/macOS/Windows collectors and shared wire fields are unchanged.

## WIFI-ABI-12 — BLOCKED, with reference exports now verified

The [symbol manifest](ios-wifi-native-abi-manifest.json) pins historical Cykey,
Theos 12.4, Procursus and Apple CF evidence with revisions and hashes. Historical
prototypes and 12.4 exports are verified within those sources, not substituted
for reference-firmware proof.

[Read-only firmware inspection](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38741425098)
on source `6b859dd6bdf4b7be457546567ba77ef5dfc4e070` verified the exact Apple
restore SHA256, iPad4,4/j85ap BuildManifest and 12.5.7/16H81 SystemVersion. The
framework is cache-only. Apple's host cache extractor produced ARM64 MobileWiFi
UUID `FA49B5A2-2578-38A0-AE5E-5DDBB37D68EA`, SHA256
`052eef70beb4e8471471c9562772ea79437dc2aba51e0cb302f054e0972009ba`.
All six candidate exports are verified on this image. The
[report](evidence/ios-wifi-12.5.7/firmware-report.json) and limited
[disassembly](evidence/ios-wifi-12.5.7/selected-functions.txt) are retained; no
firmware binary is redistributed or executed. The second research run failed
from a missing host SDK sysroot; the corrected third run succeeded. Its checksum
cache avoids further redundant image transfers.

Target ARM64 register use corroborates the historical pointer/int C ABI:
Create stores the low 32 flag bits, CopyProperty serializes its second argument,
CopyCurrentNetwork calls a MIG operation then WiFiNetworkCreate, and
GetInterfaceName returns a borrowed field without retaining. However complete
opaque-type construction, all callee bindings, reference ownership and string
contracts remain strongly inferred rather than fully verified. The extracted
CF-string annotations are unresolved and must not be treated as literal keys.
CopyDevices, CopyCurrentNetwork and GetInterfaceName dereference their first
argument without NULL checks. CopyProperty and GetSSID have NULL guards. Generic
"NULL is safe" declarations would therefore be incorrect.

Production changes: disabled parser/client and public-CF conversion, no private
function pointer calls. Tests verify pinned bytes/prototypes, strict conversion,
NULL/public-CF type handling and release paths. Further target constructor/callee
analysis is needed before an acquiring bridge can be justified; device validation
is also mandatory before activation. Export presence alone does not close ABI.

## WIFI-ASSOCIATION-12 — BLOCKED

CopyDevices reads a manager-side cached dictionary and filters interface role 0.
This does not prove current station association or list freshness. CurrentNetwork
issues `wifi_device_copy_current_network`, which is stronger evidence than a
simple local pointer lookup, but daemon cache expiry and disconnect semantics are
unknown. The presence of RSSI and RSSI_CTL_AGR strings is verified; raw dBm units,
numeric type, error sentinels and freshness are not established by those strings.
No arbitrary negative value or normalized signal metric is labeled measured dBm.

The production protocol parser independently requires powered/station/current
network plus an explicit associated claim and an OS-resolving interface name.
It accepts only integer RSSI_CTL_AGR -150..-1; shared history saturation at -128
for valid weaker values remains intact. SSID is optional UTF-8, bounded to 32
bytes; invalid/hidden/redacted SSIDs do not suppress valid signal. BSSID/password
fields are forbidden. Every failed/disconnected poll produces nil or a new map,
never a retained prior association. These tests validate conversion of simulated
claims, not the truth of a future producer's association or units.

Tests cover no/multiple devices, missing metadata, disabled/disconnected states,
wrong dictionary/number, sentinel/overflow/zero/positive/scaled values, same/new
network reconnect, invalid/hidden SSID, failure after success and recovery through
actual conversion and JSON/CBOR shared models. Physical disconnect/reconnect and
raw-unit corroboration remain mandatory. No acquisition code is activated.

## WIFI-AUTH-12 — BLOCKED

The exact firmware's MobileWiFi and wifid contain the candidate string
`com.apple.wifi.manager-access`; the pinned Procursus example requests that key
alongside broader unrelated privileges. This establishes a candidate only. It
proves neither necessity for these exact read-only calls nor sufficiency for an
ordinary LaunchDaemon/helper under Amethyst. No source compilation, root UID,
framework load or export inventory proves private daemon authorization.

The separate helper would need its own independently validated signature and
minimum entitlements because it owns daemon IPC. The Agent does not inherit a
child's entitlement. Whether either executable needs an additional privilege is
unknown; do not copy the full netctl entitlement set. P016 owns approved bundling,
ldid integration and trusted absolute executable location/ownership. P012A adds no
entitlement plist, production signing change, installer or LaunchDaemon change.
Denied-call latency and clean failure need reference-device evidence. No such
call has been executed in this milestone.

## WIFI-BOUND-12 — PASS for inactive architecture/host fixtures; iOS pending

The target disassembly confirms synchronous MIG calls but supplies no verifiable
cancellation/return bound. Direct CGO acquisition is rejected. A separate trusted
single-process executable isolates any future private IPC. The inactive Go
client uses a fixed absolute regular executable and argv, no shell, minimal
environment, discarded stderr, 16 KiB stdout cap, one active acquisition per client,
a two-second context, Unix process-group kill, 100 ms pipe WaitDelay and direct
child Wait/reap. Close prevents new reads and cancels/awaits current acquisition
under a caller deadline. Missing/denied/crashed/timed-out helpers return no snapshot.
No goroutine is abandoned per timeout; no separate polling loop is introduced.

Real fault executables exercise oversized/empty/malformed output, crashes,
nonzero exit, missing/permission-denied/non-executable/symlink paths, retained
pipes, repeated timeouts, parallel callers, cancellation, Close and recovery.
Tests check direct-child disappearance and bounded goroutine counts under race.
Deliberate orphan fixtures use a Linux test-only subreaper; production never
claims to reap unrelated descendants. The trusted native diagnostic never forks.

This is process isolation, not cancellation of a C call. It assumes normal
POSIX termination/wait semantics and a trusted helper under the same UID; it is
not a sandbox for malicious binaries or proof about uninterruptible kernel waits.
iOS process execution/group termination/reaping and A7 resource/latency behavior
still require P018. The tested architecture addresses the source-side blocker,
but does not independently authorize activation.

## Separate native diagnostic and ownership fixtures

`tools/ios-wifi-probe` is a separate disabled C executable. Its snapshot mode
always returns unavailable. Identity and inventory modes never cast or invoke a
private function. The fixed framework is optionally dlopened/dlclosed; absent
framework/exports are booleans, not startup failures. No SSID, BSSID or password
is collected. It is built into a separate review artifact, never the Agent/Hub
payload, installer or production signing. The source-selection validator rejects
any accidental ioshelper import into normal Agent/Hub dependency graphs.

Public-CF fixtures execute real conversion and owned-container cleanup under
address/undefined sanitizers, with a custom allocator counting repeated release
paths. LeakSanitizer is unavailable on hosted macOS and is not claimed; private
opaque objects are not simulated as proven CF collections. Missing/partial/all
symbol fixtures deliberately export incompatible functions which abort if called:
passing inventory proves no private invocation and does not validate signatures.
Native cross-compilation is evidence of build compatibility only.

## Read-only P018 protocol — NOT EXECUTED

1. Obtain explicit user authorization and access for a reviewed diagnostic build;
   P016 must first approve its trusted location, checksum and signing policy.
   Do not deploy or sign automatically. Retain only sanitized JSON and metadata.
2. Verify the separately recorded SHA256 and `--build-info-v1` source revision.
   Run `--identity-v1` under an externally bounded process supervisor and confirm
   iPad4,4 and build 16H81; record Amethyst/Procursus versions supplied by the user.
   No package refresh/installation or jailbreak reconfiguration is permitted.
3. Run `--symbols-v1` with the same two-second supervisor, bounded stdout and
   direct-child reaping. Record availability of the fixed framework and six
   symbols, exit code and elapsed time. These are inventories, not ABI/auth tests.
4. Run `--snapshot-v1`; this revision MUST report unavailable. This checks only
   protocol and shutdown behavior. Repeated/parallel/timeouts may test the bounded
   client without private calls; record actual PID/reap/latency, not cloud estimates.
5. **STOP before acquisition.** This tool has no association/RSSI call mode.
   First complete typed ABI/opaque ownership and authoritative station/raw-dBm
   review. Implement and review a new bounded read-only acquisition revision with
   no scan/configuration operations, then obtain authorization for that revision.
6. Only after that review, compare denied signing with the separately approved
   minimum manager-access candidate, where authorized. Record exact entitlement
   sets, per-call success/denial and latency. Do not infer sufficiency from root,
   use broader netctl rights, patch wifid/kernel/trust caches or disable enforcement.
7. In the future acquiring revision, the user alone changes Wi-Fi through normal
   controls: connected → disconnected/disabled → reconnected (same/other network).
   Require no reading after disconnect/denial, fresh recovery, real interface name,
   native raw dBm corroboration, no SIGILL, stable service and bounded resources.
   Store only transition labels, RSSI/interface, status/latency and firmware/build;
   redact all SSIDs, BSSIDs/passwords and unrelated network identifiers.

Steps 5–7 are requirements for future reviewed code, not executable commands
provided by this revision. They cannot be represented as completed tests. If ABI
or freshness cannot be established safely, keep native RSSI unavailable.

## Acceptance and handoff

Focused and final cloud results, exact SHAs, CI URLs and artifact digests are in
[ios-wifi-rssi-validation.json](ios-wifi-rssi-validation.json) under `p012a`; the
[checkpoint ledger](ios-wifi-rssi-checkpoints.md) retains all earlier evidence.
Native fixtures/ARM64 builds do not prove authorization or real RSSI measurement.
P012 BLOCKED, Gate C PENDING, physical acceptance P018/Gate F. PR #14 stays Draft;
no merge or P013 begins. No device, production database, release, production iOS
branch or frozen main is modified. Next action is the missing target ABI/daemon
association/raw-units review, followed by explicitly authorized staged P018 proof.
