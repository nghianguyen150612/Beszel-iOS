"""P023 pins every Docker Go builder to the qualified toolchain image."""

from pathlib import Path
import re
import unittest


ROOT = Path(__file__).resolve().parents[2]
ALPINE = "golang:1.27.3-alpine@sha256:d530cf60723abf1ffae0cd16bb546b379ec0550d2f3815e5b16ed69dbc00fba4"
BOOKWORM = "golang:1.27.3-bookworm@sha256:972b4110ba0f47539a975c31e52c64765e1ff19f2d74469ae4a595c6676e80a5"
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
                self.assertRegex(text, r'(?m)^RUN test "\$\(go env GOVERSION\)" = go1\.27\.3$')

    def test_procursus_test_container_uses_the_same_digest_pinned_go(self):
        workflow = (ROOT / ".github/workflows/ios-build.yml").read_text()
        self.assertIn(f"{BOOKWORM} bash -euo pipefail -c '", workflow)
        self.assertIn('test "$(go env GOVERSION)" = go1.27.3', workflow)


if __name__ == "__main__":
    unittest.main()
