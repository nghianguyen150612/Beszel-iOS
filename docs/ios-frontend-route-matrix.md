# P015 frontend route matrix

The browser harness tests the compiled frontend served by the candidate Hub, not Vite's development server. The current harness passed once against the Hub root and once through a loopback reverse proxy at `/beszel`. Both runs use disposable Hub databases, two synthetic users, and synthetic systems/telemetry. They include the failed-login UI case and produced zero uncaught page exceptions. Local acceptance and current-source hosted iOS Build, installer lifecycle, and P013 migration workflows passed at tested head `b83e179bd84b2177cd7fc8b5b0f5460d5f2411ba`.

## HTTP and SPA routes

| Route | Context | Result | Browser assertion |
| --- | --- | --- | --- |
| / | Signed out and signed in | 200 | Login form; dashboard after sign-in |
| /containers | Synthetic system owner | 200 | Synthetic container is visible |
| /smart | Synthetic system owner | 200 | Synthetic SMART device is visible |
| /monitors | Synthetic system owner | 200 | Synthetic network monitor is visible |
| /system/:synthetic-id | Synthetic system owner | 200 | System detail and charts load |
| /settings/general | Signed in | 200 | Locale can be selected and persists |
| /settings/alert-history | Signed in | 200 | Alert history route loads |
| /forgot-password | Signed out | 200 | Recovery form is visible |
| /request-otp | Signed out | 200 | OTP form is visible |
| /settings | Fresh Hub route probe | 200 | SPA index fallback |
| Nested refresh | Signed in at alert history | 200 | Reload retains nested route |
| SPA history | Signed in | PASS | Monitor navigation and browser Back work |
| Monitor details | Signed in | PASS | Dialog opens; system link returns to detail |

All listed routes returned 200 in the candidate Hub route probe and browser visits. Nine routes were checked on a fresh database at root and /beszel. JS, CSS, image, font, and locale requests loaded from embedded assets.

## Authentication and security scenarios

| Scenario | Result | Evidence |
| --- | --- | --- |
| Synthetic administrator login | PASS | Login reaches authenticated dashboard |
| Wrong password | PASS | Error toast returns to the login form; no uncaught page exception |
| Logout | PASS | User Actions returns to login |
| Second synthetic account | PASS | Outsider sees only its own system |
| Cross-user system access | PASS | Non-member system is not rendered |
| Expired session | PASS | Expired JWT is rejected; login is shown |
| Controlled failed stats request | PASS | Safe fallback UI remains |
| Account state isolation | PASS | Second account does not see first account's system |
| Unauthenticated collection reads on fresh DB | PASS | Six collections return empty lists |
| Unknown filesystem-like paths | PASS | .env, .git/config, go.mod, and missing assets return 404 |

The frontend handles expected authorization failures. Diagnostics record status only, not token, password, or response body.

## Data and chart coverage

The synthetic fixture contains a valid zero sample, populated CPU/memory/disk/network samples, container, SMART device, systemd service, alert and history, and monitor history. The fixture refreshes its synthetic chart rows on each loopback-only seed so they remain inside the default one-hour window.

The browser rendered seven system charts in RTL mode and checked their presence. Existing frontend tests provide broader source-derived contracts: Linux, Windows, macOS, and iOS API snapshots; absent, null, empty, zero, and missing optional metric handling; monitor loss-only data; and chart gaps. See internal/site/src/lib/p014-api-contracts.test.ts and internal/site/src/lib/metrics.test.ts.

There were no page exceptions. Browser diagnostics include the expected 400 from the rejected password, the intentionally aborted stats request used for fallback coverage, and 401 responses after logout/session expiry. The harness verifies that diagnostics contain no synthetic password. No screenshots were captured.

## Static Hub checks

The generated frontend had 95 assets. The Hub served 94 non-index assets with exact manifest hashes; `index.html` is dynamically modified for base path/version injection. MIME checks passed. The 17-range probe returned 400. Supported SPA paths returned 200; missing and filesystem-like paths returned 404. Final follow-up local runs passed at both root and `/beszel`. Current-source hosted iOS Build, installer lifecycle, and P013 migration fixture also passed; the built Hub artifact reports 95 embedded assets and the exact tested source revision.
