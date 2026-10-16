# P013 Hub security and authorization validation

**Evidence level:** synthetic hosted v0.19→v0.21 Hub plus local Go/API and race tests. The migration/API artifact is [run 38848300763](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38848300763), tested source `db82eb3b4d2066182292d1601b5e2e763ed05f68`. OAuth duplicate-identity and verified-email account-linking coverage passes at that source; the new email-linking test passes normally and under `-race`, as does the full Go suite. Final-head native CI passed at [run 38848296639](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38848296639). No iPad, production database, real account, or production service was used.

## First-user bootstrap

The hosted synthetic harness injects failure while saving the first superuser after the regular-user write. It observes HTTP 500, zero regular users, and exactly one `_@b.b` placeholder superuser. That placeholder remains singular across restart so setup can be retried. Two concurrent real HTTP requests then produce exactly one successful owner (`200`) and one denial (`403`); successful setup replaces the placeholder with that owner, leaving one admin user, one matching superuser, and `firstRun=false`.

Focused tests: `TestFirstUserCreation`, `TestFirstUserBootstrapAtomicity`, and hosted `first_user_bootstrap_021` manifest section.

## OAuth roles and identity validation

`TestOAuthUserRole` sends actual PocketBase OAuth auth requests with empty, `admin`, and `readonly` create data. A new OAuth account always gets `role=user`. The same provider identity logging in again after an administrator assigns a role preserves that role.

`TestOAuthInvalidIdentityDoesNotCreateAccount` uses the actual OAuth route with a missing provider ID and, separately, a missing email. Both requests fail and leave both the `users` and `_externalAuths` collections empty. These tests use synthetic provider responses and credentials only.

`TestOAuthDuplicateIdentityUsesExistingAccount` logs in through the actual route twice using the same provider ID and a different second email. The second login returns the original account ID; account and external-auth counts stay at one; the original email and administrator-assigned role remain; and attempted `createData` role escalation has no effect. `TestOAuthAccountLinkingPreservesExistingRole` also links a verified existing readonly account by OAuth email while `createData` requests admin; the existing account ID and readonly role remain and only one account/auth relation exists. Both tests pass normally and under `-race`.

## Trusted proxy identity headers

The prior middleware honored a configured trusted identity header from any peer when no proxy list was set. P013 now ignores the identity header unless the immediate peer matches an explicitly configured `TRUSTED_PROXY_IPS` address or CIDR. Ordinary PocketBase authorization still runs afterward.

`TestTrustedHeaderMiddleware` and `TestTrustedHeaderProxyAllowlist` exercise the actual API route with a synthetic user. Cases include no configured auth header; missing, empty, and blank proxy allowlists; valid IPv4 and multiple CIDRs; peer outside the list; invalid-only and mixed valid/invalid CIDRs; and a malformed comma-separated identity header. Parser tests cover IPv4, IPv6, CIDR masking, IPv4-mapped IPv6, invalid entries, empty lists, and malformed peers. Untrusted direct requests receive HTTP 401 even when they supply a valid user's email in the trusted header. Header-based authentication requires proxy addresses to be configured explicitly.

## Cache confidentiality

`TestSystemdInfoCachePolicy` sends two users' tokens and an unauthenticated request through the actual Hub handler. Authorized responses use `Cache-Control: private, max-age=60` and `Vary: Authorization`. Unauthorized responses do not contain the private service details and do not cause a request to the synthetic Agent. A shared-system request succeeds only while the system is shared.

## Monitor ownership and alert authorization

The migrated real Hub API exercise creates a regular user's monitor and alert, updates them, reads authorized lists, creates a synthetic result row, and retrieves it through monitor history. After restart, records and system/user relations remain intact. A synthetic outsider cannot read or update the monitor, result history, alert, or alert history; list queries expose no foreign records. Owner APIs delete the synthetic monitor and alert, and monitor deletion removes its result history.

Focused tests include `TestNetworkMonitorResultOwnership`, reconnect/pruning/monitor sync tests, `TestAlertRecordSystemAccess`, `TestAlertRecordForeignSystemDelivery`, stored subscription and revocation cases, alert-history access tests, and the migrated API harness. The tests cover normal writes, spoofed monitor IDs, historical and realtime updates, stale/reconnected agents, CRUD ownership, system membership, user/system reassignment, subscriptions, notification delivery after access revocation, and cross-user history reads.

## Results and limits

- The canonical Go suite passed at source `db82eb3b4d2066182292d1601b5e2e763ed05f68`. Fresh race reruns at that head passed migrations, users/OAuth, bootstrap, proxy authentication, cache policy, and monitor ownership/systems; the new verified-email OAuth link test passed normally and under `-race`. Earlier P013 focused race suites passed alerts and records/retention.
- The broader upstream Hub race command reports an existing race in `TestAgentWebSocketIntegration`, where the test reads exported `connectionManager.State` without the mutex used by the writer. P013-targeted suites pass.
- Cloud tests establish application/API behavior on Linux with synthetic data. They do not prove runtime migration or service behavior on the A7 iPad.
- Native Wi-Fi RSSI remains `DEFERRED — REFERENCE DEVICE UNAVAILABLE`; P018 owns physical validation. No release is authorized.
