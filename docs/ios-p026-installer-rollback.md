# P026 installer, backup, and rollback

**Status: NOT STARTED.** No backup, service stop, production upgrade, rollback, purge, or database operation has occurred.

## Separate approval and recovery readiness

Before modifying a running service, obtain separate explicit operator approval. Confirm physical access, stable external power, manual jailbreak reactivation, recovery SSH/terminal, adequate storage, known-good existing Agent/Hub binaries, and a complete installation inventory.

Prepare a verified cold backup of:

- /usr/local/bin/beszel-agent
- /usr/local/bin/beszel-hub
- /usr/local/bin/beszel-ios
- /Library/LaunchDaemons/dev.beszel.agent.plist
- /Library/LaunchDaemons/dev.beszel.hub.plist
- /var/lib/beszel-agent
- /var/lib/beszel-hub
- /var/lib/beszel-ios

Include relevant installation state and service configuration. Stop database writers only after explicit approval. Verify the backup is consistent and can be restored to an isolated location; keep an independent trusted copy off-device. Redact secrets from reports.

No production upgrade may proceed without verified backup and restore readiness. Do not manually overwrite the live database.

## Candidate-specific upgrade

Only after isolated Agent and Hub execution pass, present the exact reviewed procedure and obtain separate upgrade authorization. Recheck the cold backup, use the immutable P025 candidate bytes, and follow the rootful installer/service lifecycle contract.

The standard public installer resolves the published Latest release and does not select this unpublished candidate. Do not run its ordinary install/update flow as a way to install P025. Do not copy candidate binaries into /usr/local/bin before the approved transaction begins.

After an approved upgrade, verify Agent and Hub status, existing users and historical records, service restart behavior, and installer transaction state. Do not execute purge operations.

## Rollback constraints

Verify transaction rollback procedures before use. Keep binary rollback and database rollback as separate cases. Do not start an old Hub against a newly migrated database unless schema compatibility is proven. If database recovery is required, use a complete compatible backup. Stop on uncertain backup consistency, failed startup, or incomplete transaction recovery.
