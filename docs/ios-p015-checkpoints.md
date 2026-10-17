# P015 checkpoints

## Repository state

- Repository: `nghianguyen150612/Beszel-iOS`.
- Feature branch: `ios/frontend-assets-localization-security`.
- Base and starting integration SHA: `ios/upstream-v021-integration` at
  `8cd928e3cbb3e4f22556ad82146e109685ed7261`.
- P014 PR #17 is merged normally at that SHA. The abandoned PR #16 is not part
  of the integration branch.
- P014 evidence records 178/178 frozen upstream commits and 13/13 logical iOS
  patches. Production `iOS`, frozen `main`, and the P012 native RSSI deferral
  remain unchanged.

## Checkpoint A — reproducible frontend build and generated asset manifest

**Status: PASS for build reproducibility and static asset integrity. Gate D is
still pending.**

- Cleaned the ignored `internal/site/node_modules` and `internal/site/dist`
  directories before each of two installs.
- Used Go 1.27.1 and Bun 1.4.0.
- Both `bun install --frozen-lockfile` runs and both complete `bun run build`
  pipelines passed. The pipeline ran Lingui extraction, catalog compilation,
  TypeScript project build, and Vite production build.
- Both builds generated 95 non-empty files totaling 3,356,886 bytes. Their
  sorted path/size/SHA-256 manifests matched exactly with manifest digest
  `c8564be283e378bd64a282acc50e0255c5094f8793bec9ab21731b15863c7708`.
- `bun.lock` remained at
  `16643e3228f9b0fac1e5134d89782c096f49e8071489cc2a4f977f3565f02afa`.
  `package-lock.json` also remained unchanged. The reviewed CI frontend build
  workflows invoke Bun, not `npm ci`; the npm lock remains available and was
  not regenerated or removed.
- Lingui extraction added current source references and two untranslated
  Vietnamese entries. These generated `.po` changes were reviewed, then
  restored following the repository's existing CI practice. Translator text
  was preserved. Vietnamese translation coverage remains incomplete.
- The SHA-256 asset validator checks non-empty output, `index.html`, JS/CSS
  bundles, relative HTML/CSS/JS references, registered locale source modules,
  locale chunks, and exact manifest contents. Its 12 fixture tests cover valid,
  missing, empty, incomplete, stale, and unsafe-path cases.
- `internal/site` Go embed test passed against the real generated directory and
  committed SHA-256 manifest. No generated `dist` files were committed.
- Existing Vite warnings remain for overlapping dynamic/static imports and
  large WASM/vendor chunks; the build exits successfully.

The complete manifest and machine-readable checkpoint are in
[`ios-frontend-assets-manifest.sha256`](ios-frontend-assets-manifest.sha256)
and [`ios-frontend-assets-validation.json`](ios-frontend-assets-validation.json).

## Remaining work

P015 continues with actual candidate Hub serving and browser route/auth checks,
locale runtime and RTL checks, dependency advisory triage, full regression and
race suites, and hosted final-head iOS CI. No Gate D pass is claimed here.
