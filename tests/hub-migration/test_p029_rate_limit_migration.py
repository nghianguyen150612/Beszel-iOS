"""Focused P029 synthetic-database checks for the additive rate-limit migration."""

import copy
import unittest

from harness import (
    HarnessError,
    P029_RATE_LIMIT_MIGRATION,
    P029_RATE_LIMIT_STATE,
    validate_p027_candidate_delta,
)


class P029RateLimitMigrationTests(unittest.TestCase):
    def setUp(self):
        self.p027 = {
            "schema_sha256": "schema-before",
            "record_counts": {"users": 2, "systems": 1},
            "application_settings": {"appName": "Synthetic", "appURL": "https://synthetic.invalid"},
            "migration_ledger": [["p027-a.go", 10], ["p027-b.go", 20]],
            "rate_limit_settings": {"enabled": False, "rules": [{"label": "*:auth", "maxRequests": 2, "duration": 3}]},
        }
        self.candidate = copy.deepcopy(self.p027)
        self.candidate["migration_ledger"].append([P029_RATE_LIMIT_MIGRATION, 30])
        self.candidate["rate_limit_settings"] = copy.deepcopy(P029_RATE_LIMIT_STATE)

    def test_expected_p029_migration_is_the_only_ledger_delta(self):
        validate_p027_candidate_delta(self.p027, self.candidate)

    def test_existing_rows_schema_records_and_settings_must_be_preserved(self):
        for key, value in (
            ("schema_sha256", "changed-schema"),
            ("record_counts", {"users": 1}),
            ("application_settings", {"appName": "changed"}),
        ):
            candidate = copy.deepcopy(self.candidate)
            candidate[key] = value
            with self.subTest(key=key), self.assertRaisesRegex(HarnessError, f"P029 migration changed P027 {key}"):
                validate_p027_candidate_delta(self.p027, candidate)

        candidate = copy.deepcopy(self.candidate)
        candidate["migration_ledger"][0][1] = 99
        with self.assertRaisesRegex(HarnessError, "changed or reordered an existing P027"):
            validate_p027_candidate_delta(self.p027, candidate)

    def test_unexpected_ledger_entries_or_rate_limit_settings_fail(self):
        candidate = copy.deepcopy(self.candidate)
        candidate["migration_ledger"].append(["unexpected.go", 40])
        with self.assertRaisesRegex(HarnessError, "ledger delta is not"):
            validate_p027_candidate_delta(self.p027, candidate)

        candidate = copy.deepcopy(self.candidate)
        candidate["rate_limit_settings"]["rules"][0]["maxRequests"] = 11
        with self.assertRaisesRegex(HarnessError, "exact sign-in-only"):
            validate_p027_candidate_delta(self.p027, candidate)


if __name__ == "__main__":
    unittest.main()
