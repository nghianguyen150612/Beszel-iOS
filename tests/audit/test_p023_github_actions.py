"""P023 static supply-chain and workflow-boundary regression checks."""

from __future__ import annotations

import re
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
WORKFLOWS = ROOT / ".github/workflows"
ACTION_REF = re.compile(r"^\s*(?:-\s*)?uses:\s*([^\s#]+)")
COMMIT_SHA = re.compile(r"^[0-9a-f]{40}$")


class P023GitHubActionsTests(unittest.TestCase):
    def test_all_external_actions_are_pinned_to_commit_shas(self):
        uses = []
        for workflow in sorted(WORKFLOWS.glob("*.yml")):
            for number, line in enumerate(workflow.read_text().splitlines(), start=1):
                match = ACTION_REF.match(line)
                if not match:
                    continue
                reference = match.group(1)
                if reference.startswith("./"):
                    continue
                self.assertIn("@", reference, f"{workflow.name}:{number}: {reference}")
                _repository, revision = reference.rsplit("@", 1)
                self.assertRegex(
                    revision,
                    COMMIT_SHA,
                    f"{workflow.name}:{number}: external action is not pinned to a full commit SHA: {reference}",
                )
                uses.append(reference)
        self.assertGreaterEqual(len(uses), 40, "workflow scan unexpectedly covered too few action steps")

    def test_publish_job_alone_has_helm_registry_write_permission(self):
        source = (WORKFLOWS / "helm-charts.yml").read_text()
        jobs = source.split("jobs:\n", 1)[1]
        validation = jobs.split("  publish:\n", 1)[0]
        publish = jobs.split("  publish:\n", 1)[1]
        self.assertNotIn("packages: write", source.split("jobs:\n", 1)[0])
        self.assertNotIn("packages: write", validation)
        self.assertIn("packages: write", publish)
        self.assertIn("if: github.event_name == 'push'", publish)
        self.assertIn("persist-credentials: false", publish)

    def test_checkout_write_jobs_use_short_lived_askpass_credentials(self):
        cases = {
            "sync-upstream-main.yml": "GIT_PUSH_TOKEN: ${{ secrets.UPSTREAM_SYNC_TOKEN }}",
            "update-helm-charts.yml": "GIT_PUSH_TOKEN: ${{ env.AUTOMATION_TOKEN }}",
        }
        for name, token_env in cases.items():
            with self.subTest(workflow=name):
                source = (WORKFLOWS / name).read_text()
                self.assertIn("persist-credentials: false", source)
                self.assertIn(token_env, source)
                self.assertIn("GIT_ASKPASS=\"$askpass\"", source)
                self.assertIn("GIT_TERMINAL_PROMPT=0", source)

    def test_release_builds_disable_reusable_dependency_caches(self):
        release = (WORKFLOWS / "release.yml").read_text()
        docker = (WORKFLOWS / "docker-images.yml").read_text()
        ios = (WORKFLOWS / "ios-build.yml").read_text()
        self.assertIn("bun-version: 1.4.0", release)
        self.assertIn("no-cache: true", release)
        self.assertIn("cache: false", release)
        self.assertIn("bun-version: 1.4.0", docker)
        self.assertIn("no-cache: true", docker)
        self.assertNotRegex(ios, re.compile(r"(?m)^\s+cache: true$"))
        self.assertIn("cache: false", ios)
        self.assertIn("no-cache: true", ios)

    def test_chart_package_output_is_not_interpolated_into_shell_source(self):
        source = (WORKFLOWS / "helm-charts.yml").read_text()
        self.assertIn("CHART_PACKAGE: ${{ steps.package.outputs.package }}", source)
        self.assertIn('run: helm push "$CHART_PACKAGE"', source)
        self.assertNotIn('run: helm push "${{ steps.package.outputs.package }}"', source)

    def test_hosted_workflow_scanners_are_digest_pinned_and_upload_raw_results(self):
        source = (WORKFLOWS / "p017-release-candidate.yml").read_text()
        self.assertIn(".github/workflows/**", source)
        self.assertIn("zizmor/releases/download/v1.31.1/", source)
        self.assertIn("d185f726bdaf92df78273217dfdd24ebcd6b0ef48cf3ce776ff5762b6e138c99", source)
        self.assertIn("actionlint/releases/download/v1.7.12/", source)
        self.assertIn("8aca8db96f1b94770f1b0d72b6dddcb1ebb8123cb3712530b08cc387b349a3d8", source)
        self.assertIn("--strict-collection --no-config --no-ignores", source)
        self.assertIn("name: p023-workflow-security-evidence", source)
        self.assertIn("p023-zizmor.json", source)
        self.assertIn("p023-actionlint.log", source)


if __name__ == "__main__":
    unittest.main()
