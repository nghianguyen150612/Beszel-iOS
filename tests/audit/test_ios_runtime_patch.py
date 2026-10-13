"""Exercise the A7 patch against a frozen Go 1.27.1 runtime source fragment."""

import os
from pathlib import Path
import subprocess
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[2]
PATCH = ROOT / ".github/scripts/patch-go-ios-arm64-runtime.py"
FIXTURE = Path(__file__).with_name("fixtures") / "go1.27.1-procyield-arm64.s"
START = "TEXT runtime·procyieldAsm(SB),NOSPLIT,$0-0"
END = "\n// Save state of caller into g->sched,"


class RuntimePatchTests(unittest.TestCase):
    def run_patch(self, source, success):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            asm = root / "src/runtime/asm_arm64.s"
            asm.parent.mkdir(parents=True)
            asm.write_text(source)
            fake_go = root / "go"
            fake_go.write_text('#!/bin/sh\nprintf "%s\\n" "$FIXTURE_GOROOT"\n')
            fake_go.chmod(0o755)
            env = dict(os.environ, PATH=f"{root}:{os.environ['PATH']}", FIXTURE_GOROOT=str(root))
            result = subprocess.run(["python3", str(PATCH)], env=env, capture_output=True, text=True)
            self.assertEqual(result.returncode == 0, success, result.stdout + result.stderr)
            output = asm.read_text()
            if success:
                body = output[output.index(START):output.index(END)]
                self.assertIn("YIELD", body)
                self.assertNotIn("CNTVCT_EL0", body)
                self.assertEqual(output.split(START)[0], source.split(START)[0])
                self.assertEqual(output.split(END)[1], source.split(END)[1])
            else:
                self.assertEqual(output, source, "unknown layouts must fail without writing")
            return output

    def test_known_go_1_27_1(self):
        self.run_patch(FIXTURE.read_text(), True)

    def test_missing_start(self):
        self.run_patch(FIXTURE.read_text().replace(START, "unknown function"), False)

    def test_duplicate_start(self):
        self.run_patch(FIXTURE.read_text() + START, False)

    def test_missing_end(self):
        self.run_patch(FIXTURE.read_text().replace(END, "\nunknown end"), False)

    def test_unknown_counter_instruction(self):
        self.run_patch(FIXTURE.read_text().replace("CNTVCT_EL0", "UNKNOWN_COUNTER"), False)

    def test_already_patched_fails_closed(self):
        self.run_patch(self.run_patch(FIXTURE.read_text(), True), False)


if __name__ == "__main__":
    unittest.main()
