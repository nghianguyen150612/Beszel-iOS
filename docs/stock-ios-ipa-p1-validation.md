# Stock iOS IPA — P1 hosted qualification

## Exact-source checkpoint

- Branch: `feature/stock-ios-ipa` (based on `iOS` `0c3e4cd22ac24264834c401ad412e44aeda4e4f3`).
- Qualified source: `9d2fb893feceb6f53522343bd0a7ba86eb0a7dee`.
- Draft PR: [#35](https://github.com/nghianguyen150612/Beszel-iOS/pull/35), base `iOS`.
- Workflow: [Stock iOS IPA run 45332869058](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/45332869058) — **PASS**.
- macOS runner: `macos-15`; observed Xcode `16.4`, iPhoneOS SDK `18.5`.
- Go: `1.27.1`. No patched Apple A7 runtime is used for this stock A15-targeted proof.

The first run, [45332705938](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/45332705938), built and linked Go/Swift, produced a Mach-O arm64 executable and passed ad-hoc code-signature verification, but **FAIL**ed the final SHA256SUMS check because it ran outside the artifact directory. Commit `9d2fb893` corrects only the checksum working directory; the second run completed **PASS**.

## Artifact provenance

| Item | Value |
| --- | --- |
| Actions artifact | `beszel-stock-ios-development-ipa` |
| Artifact ID | `14891644417` |
| Uploaded ZIP SHA-256 | `4e9b7dd5b56823bb5e304eb910c5133494371db6447f93146d560e2294b618c8` |
| Embedded IPA | `BeszelStockAgent-development-adhoc.ipa` |
| IPA SHA-256 | `4735c4275b640b262f1e0103fdd306b7672f62019f3321dfe8bcc99056f36fb0` |
| IPA size | `914624` bytes |
| Binary architecture | Mach-O 64-bit arm64, file type `MH_EXECUTE` |
| Mach-O platform | `PLATFORM_IOS` (2) |
| Mach-O minimum OS | iOS `16.0` |
| Mach-O build SDK | iPhoneOS `18.5` |
| Bundle ID | `dev.nghianguyen.beszel.stockagent` |
| Bundle version | `0.1.0 (1)` |
| Binary evidence | Go `go1.27.1` and `BeszelGoVersion` symbol present |

The uploaded ZIP and inner IPA passed independent ZIP integrity checks. The IPA's SHA-256 matches the attached `SHA256SUMS` exactly. Its `Info.plist` parses cleanly and declares iPhoneOS, iOS 16.0 minimum, and iPhone device family. Hosted `codesign --verify` passed with an **ad-hoc** signature.

## Meaning of PASS

P1 **build and native-link qualification: PASS**. This is an actual iPhoneOS executable statically linked with a Go runtime and the shared repository's Beszel version. The SwiftUI source calls Go-native exports directly; it does not shell out to a helper binary.

It is **not** yet a functional Beszel Agent: `agent.NewAgent` is not started, no Hub registration/WebSocket has been attempted, no stock metrics are collected and the UI deliberately labels these P2 items as inactive. The existing jailbroken Agent/Hub binary build remains untouched.

The IPA is ad-hoc signed for development packaging; it requires valid device-specific re-signing/provisioning through SideStore or another permitted signing method. This artifact is **not** a signed stock-device installation success, and no physical iPhone 13 test was performed.

## P2/P3 follow-up

1. Extract an app-controlled Agent lifecycle from the existing `agent/agent.go` and `agent/connection_manager.go`, with cancel/Stop suitable for foreground sessions. Add regressions for the original jailbreak path.
2. Reuse `agent/client.go`'s outbound WebSocket/CBOR protocol, and bridge hub configuration to Keychain-backed Swift storage without persisting tokens in plaintext.
3. Implement public-API battery, memory, device identity and container-storage collectors. Mark truly unavailable metrics unsupported rather than fabricating zero.
4. Add foreground lifecycle, disconnect/reconnect and background suspension handling without a daemon, private API, exploit or misused background mode.
5. Test against a disposable Hub, then (only with fresh authorization) validate a re-signed build on the specified stock iPhone 13.

## Restrictions

No merge, release, production access, jailbreak service changes or device operations are authorized. The P030 integration stack remains independent. Stock iPhone execution is **NOT VALIDATED / DEFERRED**.

## P2 continuation

The P1 artifact and its qualification above remain historical evidence. Subsequent
P2 source embeds the real shared Agent and adds explicit foreground lifecycle,
Keychain and native metric snapshots. Its transport is tested with both a fixture
and the actual disposable Hub handler/database. Native stats acceptance remains
BLOCKED; see [P2 handoff](stock-ios-ipa-p2-handoff.md) for current scope, new build
evidence and remaining acceptance criteria. No physical execution is claimed.
