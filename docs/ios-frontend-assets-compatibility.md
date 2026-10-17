# P015 frontend assets and compatibility

## Result

The P015 merge is recorded at integration commit `50546d7f7468f7b07af347eb157139208b04fcbc`. Follow-up source commit `60df517928b74ac3f0230dbe0d72100bbd952d94` closes a dependency advisory and tightens login/catalog/browser acceptance. Local checks pass for that follow-up; hosted CI and artifact verification for it are pending. The earlier hosted artifacts apply to the prior merged source only.

The candidate uses Go 1.27.1 and Bun 1.4.0. Two clean install/build cycles removed ignored node_modules and dist before frozen Bun installation. Both ran Lingui extraction/compilation, TypeScript, and Vite production build and produced identical sorted manifests.

- 95 non-empty files; 3,357,403 bytes.
- Manifest SHA-256: 63dc41212f5029b2cb07cfbda5f3ddaa4ea71b72f32e39f0032f870c06286df4.
- Bun lock SHA-256: e2d090fa8b5bcffd8e2de003262cfe6c506c297a676279fbb08d1c2db3bc6dbf; frozen installs left it unchanged.
- Retained npm lock SHA-256: 6fbd1027efc8368462a78a0ff87dae23b758eee759ba56172eb862f70df1b3d5; unchanged. CI uses Bun, not npm ci.
- All 32 registered locale modules compile and are included in generated assets.
- Lingui reported 537 source messages. Vietnamese has 529 translated and 8 missing; translator text was not changed.

The complete sorted manifest is in `ios-frontend-assets-manifest.sha256`. The asset validator checks non-empty output, index, JS/CSS references, locale modules/chunks, safe relative paths, and stale or incomplete manifests. It passed against the follow-up dist. The Go embedded-asset test passed against that dist and manifest.

## Candidate Hub serving

A Linux Hub was started with a fresh disposable PocketBase database. A second disposable Hub was checked at the /beszel prefix.

- 94 of 95 files matched by exact SHA-256. The Hub intentionally modifies index.html to inject version and base-path configuration.
- MIME checks passed: 88 JavaScript, 1 CSS, 2 WOFF2 fonts, 1 PNG, 1 SVG, and 1 JSON.
- Nine supported SPA routes returned 200 at root and /beszel.
- Both base paths injected the expected BASE_PATH.
- Unknown paths including .env, .git/config, go.mod, and missing assets returned 404.
- Unauthenticated collections on the fresh database returned empty records.
- Seventeen byte ranges returned 400; up to sixteen are accepted.
- Static assets returned the expected public cache policy.

The current browser harness passed at both the Hub root and `/beszel` behind the loopback reverse proxy. It used synthetic accounts and disposable data only. No physical iPad was used. Route and auth details are in `ios-frontend-route-matrix.md`.

## Focused fixes

The frontend refreshes authentication only when a token exists, clears expired or forbidden auth state, and avoids printing response bodies or tokens in browser logs. User settings are created only after a real 404; expected 401 and 403 responses are handled without logging sensitive payloads. Hub-info failures are caught and logged with status only.

The Hub disables HTTP/2 server negotiation on its production TLS server and sets the Go 1.27.1 HTTP/2 client default off in Hub and Agent executables. A TLS test offers h2 and HTTP/1.1 with GODEBUG=http2server=1 and confirms HTTP/1.1 is negotiated. Hub middleware limits byte ranges to sixteen; unit and live HTTP checks cover the boundary.

Existing Vite warnings remain for overlapping dynamic/static imports and oversized WASM/vendor chunks. The build succeeds.

## Gate status

Follow-up Gate D status: **pending current-source hosted CI and artifact verification**. The previous merged P015 source passed its hosted checks; those results do not cover commit `60df517`. The high dependency residuals remain documented in `ios-dependency-advisory-review.md`. P018 physical acceptance remains pending, and no release is authorized.
