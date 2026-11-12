"""P023 pins every Docker Go builder to the qualified toolchain image."""

from pathlib import Path
import re
import unittest


ROOT = Path(__file__).resolve().parents[2]
ALPINE = "golang:1.27.4-alpine@sha256:0a5f7fe796599ff487605987432e7619ae45beeb2fba4c2ce0b907879932aebf"
BOOKWORM = "golang:1.27.4-bookworm@sha256:4fa7c968ba93f1635670522dbb01a2c78150443e1385746924566079f36570ea"
BUILDERS = {
    "dockerfile_hub": ALPINE,
    "dockerfile_agent": ALPINE,
    "dockerfile_agent_alpine": ALPINE,
    "dockerfile_agent_intel": ALPINE,
    "dockerfile_agent_nvidia": BOOKWORM,
    "dockerfile_agent_nvidia_slim": BOOKWORM,
}


class DockerToolchainTests(unittest.TestCase):
    def test_all_go_builder_images_are_digest_pinned_and_version_checked(self):
        for name, image in BUILDERS.items():
            with self.subTest(dockerfile=name):
                text = (ROOT / "internal" / name).read_text()
                self.assertRegex(text, rf"(?m)^FROM --platform=\$BUILDPLATFORM {re.escape(image)} AS builder$")
                self.assertRegex(text, r"(?m)^ENV GOTOOLCHAIN=local$")
                self.assertRegex(text, r'(?m)^RUN test "\$\(go env GOVERSION\)" = go1\.27\.4$')

    def test_procursus_test_container_uses_the_same_digest_pinned_go(self):
        workflow = (ROOT / ".github/workflows/ios-build.yml").read_text()
        self.assertIn(f"{BOOKWORM} bash -euo pipefail -c '", workflow)
        self.assertIn('test "$(go env GOVERSION)" = go1.27.4', workflow)


if __name__ == "__main__":
    unittest.main()
