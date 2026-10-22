# P025 Security Disposition

**Status: P024 baseline recorded — P025 exact-head scans pending.**

Baseline disposition reported by P024:

- `GO-2026-6608` and `GO-2026-6809`: remediated under Go 1.27.3; re-scan the exact P025 source.
- `GO-2026-5932`: module-level finding; P024 reports no demonstrated OpenPGP package reachability in reviewed profiles. Recheck reachability and mitigation evidence.
- `GHSA-vfj7-8cjw-p6xm`: HIGH build-only `braces` dependency finding under the documented P023 mitigation. Recheck the pinned dependency and build-only scope.

These remain inherited observations, not P025 scan results or release-owner risk acceptance. The hosted P017 workflow must rerun the exact P025 source scans. Any new reachable critical issue blocks qualification. Release-owner decisions remain for P027/P028.
