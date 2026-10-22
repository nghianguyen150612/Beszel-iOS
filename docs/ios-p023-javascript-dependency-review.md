# P023 JavaScript dependency review

**Qualified implementation source:** fd6b1595ff2e8b4d7037229ce6d71af87f907b4e
**Reviewed on:** 2026-10-22
**Finding:** GHSA-vfj7-8cjw-p6xm / CVE-2026-93687

## Lockfile graph and runtime boundary

Both internal/site/bun.lock and internal/site/package-lock.json resolve braces 3.0.3. The transitive paths are:

- @lingui/vite-plugin → @lingui/cli → chokidar → braces
- @lingui/vite-plugin → @lingui/cli → micromatch → braces

The Lingui CLI/plugin and braces are development dependencies. Bun resolves Lingui CLI 5.4.1; npm resolves 5.9.5. P023 checks both exact lock graphs.

The current GitHub Advisory/OSV record describes HIGH severity (CVSS 3.1 score 7.5), affects braces through 3.0.3, and lists no fixed version. The Hub embeds generated static assets and does not ship or execute Node, Lingui, chokidar, micromatch, or braces at runtime.

The affected package processes patterns during frontend build. Current patterns come from checked-in Lingui/Vite configuration, not runtime Hub API input. A contributor able to change that configuration in a PR can still cause build-time denial of service.

## Audit evidence

- Bun 1.4.0 used bun install --frozen-lockfile; the lockfile remained unchanged.
- Bun audit retains the reviewed high-severity GHSA.
- The latest npm audit completed successfully as a scan and reported five high-severity package entries: @lingui/cli, @lingui/vite-plugin, chokidar, micromatch, and braces. All five trace to the same braces GHSA.
- npm's fixAvailable proposal is a major Lingui CLI/plugin 6.10.0 upgrade. It is not a published fixed braces release and was not treated as remediation.
- The independent source-bound OSV scan covers all 510 package-lock nodes and 494 unique exact package/version queries. It reports only GHSA-vfj7-8cjw-p6xm for braces 3.0.3 and no fixed version.
- npm audit completed, so the full-lock OSV scan is an independent cross-check rather than a fallback on this source. An earlier exact registry timeout and its raw evidence remain historical; only that exact error permits fallback.
- Raw Bun, npm, OSV query, OSV response, and summary files are retained in the P017 JavaScript audit artifact.

The hosted scan artifact 12400546337 has GitHub SHA-256 50c62882b601152bbb34672030300375ba1ae89d3b93566ee9e47000144ee9cf. P017's JavaScript scan step passed its fail-closed exact graph and advisory policy on source fd6b1595ff2e8b4d7037229ce6d71af87f907b4e.

## Mitigation and residual risk

No compatible fixed package was available on 2026-10-22. P023 did not apply the proposed major Lingui upgrade because no verified dependency graph showed it removes the affected braces path. No translation catalog or language was removed.

PR builds use isolated hosted runners, read-only repository permissions, checkout with persisted credentials disabled, no release secrets, and bounded timeouts. P023 qualifies frozen installation, Bun tests, Lingui extraction/compilation, TypeScript/Vite production build, localization, asset embedding, and lockfile immutability. Vietnamese, Brazilian Portuguese, and Catalan catalogs remain present.

Disposition: MITIGATED_WITH_RESIDUAL_RISK. This is a build-only mitigation; vulnerable braces remains in both lockfiles. The warning remains open for a release owner.
