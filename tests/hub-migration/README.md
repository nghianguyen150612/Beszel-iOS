# Disposable Hub migration harness

`harness.py` builds the real Beszel Hub from the pinned production v0.19 source and the v0.21 source at the clean checkout `HEAD`. It builds each version's actual production frontend from the frozen Bun lockfile, then starts v0.19 in a fresh temporary data directory and seeds synthetic accounts, settings, systems, and historical records through Hub/PocketBase APIs. It reopens that original fixture with the same v0.19 binary and records a redacted manifest. The SQLite database and private SSH key are never copied to the output directory.

Requirements are the Go version pinned by `go.mod` (Go 1.27.2 for the P019 candidate), Bun 1.4.0, Python 3, `git`, `tar`, and `ssh-keygen`. The checkout must be clean and descend from the verified P013 starting integration SHA `d6239915e404668c10453ab2ec44f797b2f0178e`.

Run from the repository root:

```sh
python3 tests/hub-migration/harness.py --output /tmp/p013-fixture-evidence
```

The output contains `fixture-manifest.json` with synthetic record IDs and relations, collection schema, migration ledger, counts, hashes, timestamps, and public-key fingerprint, plus sanitized failure diagnostics when migration/startup failpoints run. The full database archive and content-hashed inventory stay private to the disposable job and are never uploaded. All binaries, fixture data, synthetic credentials, private key material, logs, and generated frontend files are created beneath separate temporary directories and removed at exit. The harness refuses to operate under `/var/lib/beszel-hub`.

The full workflow also creates and verifies the cold checkpoint, restores and opens the original v0.19 state, migrates a clone with v0.21, checks schema/data/ledger/idempotence and API behavior, then verifies complete old-binary/data rollback and recovery from five injected failures. CI uploads only the redacted manifest and sanitized diagnostics; it does not upload the database archive or private key material. Hosted Linux results do not prove an on-device iOS migration.
