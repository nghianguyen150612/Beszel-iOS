# P023 JavaScript dependency review

**Reviewed candidate code commit:** `f7a4525aeb7513163439b56b330ddfbd527c4da8`\
**Reviewed on:** 2026-10-22  
**Finding:** `GHSA-vfj7-8cjw-p6xm` / `CVE-2026-93687`

## Lockfile graph and runtime boundary

Both `internal/site/bun.lock` and `internal/site/package-lock.json` resolve
`braces@3.0.3`. The transitive paths are:

```text
@lingui/vite-plugin -> @lingui/cli -> chokidar -> braces
@lingui/vite-plugin -> @lingui/cli -> micromatch -> braces
```

The Lingui CLI/plugin and `braces` are development dependencies. Bun resolves
Lingui CLI 5.4.1; npm resolves 5.9.5. P023 checks both paths against their
exact lockfile structure.

The current GitHub Advisory/OSV record describes high severity (CVSS 3.1
score 7.5), affects `braces` through 3.0.3, and has no fixed-version event.
The Hub embeds generated static assets and does not ship or execute Node,
Lingui, chokidar, micromatch, or braces at runtime.

The affected package processes patterns during the frontend build. Current
patterns come from checked-in Lingui/Vite configuration, not runtime Hub API
input. A contributor able to change that configuration in a PR can still
cause build-time denial of service.

## Audit evidence and network fallback

- Bun 1.4.0 installation used `bun install --frozen-lockfile`; the lockfile
  remained unchanged.
- Bun audit's raw report retained only the reviewed high-severity GHSA.
- On the earlier `238ef2b` workflow run, npm audit timed out at the official
  registry. P023 retained the raw timeout JSON and stderr, then used the
  full-lock OSV scan as its narrowly permitted fallback.
- The exact `7f980ea` npm audit subsequently completed. Its raw report contains
  five high-severity package entries (`@lingui/cli`, `@lingui/vite-plugin`,
  `chokidar`, `micromatch`, and `braces`) that all trace to the single reviewed
  braces GHSA. The P023 validator checks the exact package keys, install nodes,
  severity totals, graph edges, and advisory source; a new node or advisory
  fails. npm's `fixAvailable` proposal is a major Lingui CLI/plugin 6.10.0
  upgrade, not a published fixed `braces` release.
- An independent OSV query scans every exact package/version node in
  `package-lock.json`. The lock has 510 package nodes and 494 unique queries;
  the `c9e68e1` source-bound result contains only
  `GHSA-vfj7-8cjw-p6xm` for `braces@3.0.3` and lists no fixed version.
  Missing expected findings, new advisories, a stale lock hash, changed
  source SHA, or changed affected range fail closed.
- Any npm error other than the configured registry timeout fails. Raw Bun,
  npm, OSV query, OSV response, and summary artifacts are retained.

## Mitigation and residual risk

No compatible fixed package was available on 2026-10-22. P023 did not apply
the proposed major Lingui upgrade because no verified dependency graph showed
that it removes the affected `braces` path. No translation catalog or
language was removed.

PR builds use isolated hosted runners, read-only repository permissions,
checkout with persisted credentials disabled, no release secrets, and
bounded timeouts. P023 qualifies Bun tests, Lingui
extraction/compilation, TypeScript/Vite production build, localization,
asset embedding, and lockfile immutability. Vietnamese, Brazilian Portuguese,
and Catalan catalogs remain present.

Disposition: `MITIGATED_WITH_RESIDUAL_RISK`. This is a build-only mitigation;
the vulnerable package remains in both lockfiles. The warning remains open
for a release owner.
