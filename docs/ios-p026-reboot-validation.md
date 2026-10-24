# P026 reboot and persistence validation

**Status: NOT AUTHORIZED / NOT RUN.** Reboot is a distinct operation requiring separate explicit operator approval. No remote or local reboot has been attempted.

## Preconditions

Before reboot:

- All required physical acceptance cases before reboot pass.
- A verified recovery backup exists and its restore procedure is understood.
- The operator has physical access and reliable power.
- The manual Amethyst reactivation procedure is known and available.
- Recovery SSH or physical terminal access is expected after reactivation.
- The operator explicitly approves the reboot.

Do not reboot remotely unless the operator explicitly accepts the recovery risk. The iPad is semi-untethered; manual jailbreak reactivation after a full reboot is expected and is not itself an application failure.

## Post-reboot checks

After the operator manually reactivates Amethyst:

1. Verify SSH or local terminal access.
2. Verify Agent and Hub LaunchDaemon registration.
3. Verify Agent and Hub process state.
4. Verify Hub health.
5. Verify Agent reconnection.
6. Verify telemetry timestamps advance.
7. Verify configuration and historical data are preserved.
8. Check for unexpected crashes or service loops.

Capture sanitized outputs and times. Do not include credentials, database contents, personal telemetry, IP addresses, or unredacted logs. A reboot result is PENDING until every applicable check has actual device evidence.
