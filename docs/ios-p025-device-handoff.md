# P026 Physical Validation Handoff

**Status: PLAN ONLY — no device access in P025.** P018 Gate F remains **PENDING**.

Reference acceptance target:

- iPad mini 2 A1489, Apple A7, iOS 12.5.7
- Amethyst / Procursus rootful jailbreak

Candidate source SHA, tree SHA, native workflow run, immutable artifact IDs, Agent/Hub SHA-256 values, checksum manifest, validation manifest, toolchain identity, and artifact reference will be filled only after final qualification. This checkpoint is not an installable candidate.

The read-only P026 plan will cover artifact transfer; operator-managed trust/signing prerequisites; Agent and Hub startup; installer and LaunchDaemon lifecycle; reboot/jailbreak reactivation; telemetry; Agent-to-Hub, WebSocket, SSH, and TLS behavior; historical migration; backup and recovery; uninstall/rollback; and final operator review. It will identify installation paths, configuration, service controls, controlled rollback steps, and safety warnings from the verified repository payload.

No Apple trust, signing acceptance, or physical launchability claim is made here.
