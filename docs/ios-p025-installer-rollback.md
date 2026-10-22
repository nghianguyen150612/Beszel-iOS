# P025 Installer, Migration, and Rollback Qualification

**Status: P024 evidence recorded — P025 final-head revalidation pending.**

P024 run [39759219620](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39759219620) reports installer lifecycle success on its exact source head. P024 run [39759219686](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39759219686) reports the Hub migration fixture succeeded. These inherited results do not constitute P025 final-head qualification.

The P025 PR paths will rerun the existing synthetic installer lifecycle and disposable migration fixtures on the candidate commit. The acceptance matrix covers fresh install, component selection, update, repair, reconfigure, restart, backup, rollback, interrupted updates, integrity and startup failures, low disk, uninstall/purge, historical identity preservation, ordered migrations, restart idempotence, interrupted migration recovery, and cold PocketBase backup restoration. Binary rollback and database rollback remain separate. No physical install is permitted.

**Results:** pending.
