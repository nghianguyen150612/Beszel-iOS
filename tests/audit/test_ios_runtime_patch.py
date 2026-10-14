"""Disposable Go 1.27.1 fixtures; never patch the installed toolchain."""

import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

ROOT = Path(__file__).resolve().parents[2]
PATCH = ROOT / ".github/scripts/patch-go-ios-arm64-runtime.py"
FIXTURE = Path(__file__).with_name("fixtures") / "go1.27.1-procyield-arm64.s"
spec = importlib.util.spec_from_file_location("runtime_patch", PATCH)
patch = importlib.util.module_from_spec(spec)
spec.loader.exec_module(patch)


class RuntimePatchTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.asm = self.root / "src/runtime/asm_arm64.s"
        self.asm.parent.mkdir(parents=True)
        self.source = FIXTURE.read_bytes()
        self.asm.write_bytes(self.source)
        self.asm.chmod(0o640)
        self.unrelated = self.asm.with_name("unrelated.s")
        self.unrelated.write_bytes(b"must remain byte-identical\x00\xff")
        self.fake = self.root / "go"
        self.fake.write_text('#!/bin/sh\n[ "$*" = "env -json GOROOT GOVERSION" ] || exit 99\nprintf "%s\\n" "$FIXTURE_GO_ENV"\n')
        self.fake.chmod(0o755)
        self.info = {"GOROOT": str(self.root), "GOVERSION": "go1.27.1"}

    def run_patch(self, success=False):
        before = self.asm.read_bytes()
        files = {str(p.relative_to(self.root)): p.read_bytes()
                 for p in self.root.rglob("*") if p.is_file()}
        env = dict(os.environ, PATH=str(self.root), FIXTURE_GO_ENV=json.dumps(self.info))
        result = subprocess.run([sys.executable, str(PATCH)], env=env, capture_output=True, text=True)
        self.assertEqual(result.returncode == 0, success, result.stdout + result.stderr)
        after = {str(p.relative_to(self.root)): p.read_bytes()
                 for p in self.root.rglob("*") if p.is_file()}
        self.assertEqual(set(files), set(after), "no extra files may survive")
        for name, content in files.items():
            if name != "src/runtime/asm_arm64.s" or not success:
                self.assertEqual(after[name], content, name)
        if success:
            expected = before.decode().replace(
                before.decode()[before.decode().index(patch.START):before.decode().index(patch.END)],
                patch.REPLACEMENT).encode()
            self.assertEqual(self.asm.read_bytes(), expected)
            self.assertEqual(json.loads(result.stdout)["go"], "go1.27.1")
            self.assertEqual(self.asm.stat().st_mode & 0o777, 0o640)
        else:
            self.assertIn("ERROR:", result.stderr)
        return result

    def test_exact_known_go_1_27_1(self):
        self.run_patch(True)

    def test_unknown_shapes_are_byte_identical(self):
        source = self.source.decode()
        mutations = {
            "unexpected instruction": source.replace("\tRET", "\tNOP\n\tRET"),
            "different arithmetic": source.replace("MUL     R1, R0, R0", "MUL     R2, R0, R0"),
            "missing start": source.replace(patch.START, "unknown"),
            "duplicate start": source + '\n' + patch.START,
            "embedded start": source.replace(patch.START, 'prefix' + patch.START),
            "missing end": source.replace(patch.END, "\nunknown"),
            "altered end": source.replace("g->sched,", "g->other,"),
            "duplicate end": source + patch.END,
            "missing frequency": source.replace("\tMRS     CNTFRQ_EL0, R1\n", ""),
            "altered counter": source.replace("CNTVCT_EL0, R2", "CNTVCT_EL0, R3"),
            "extra counter": source.replace("delay:", "MRS CNTVCT_EL0, R1\ndelay:"),
            "missing counter": source.replace("\tMRS     CNTVCT_EL0, R1\n", ""),
            "unrelated function inside boundary": source.replace("delay:", "TEXT unrelated(SB),$0\ndelay:"),
            "truncated": source[:source.index("delay:")],
            "invalid utf8": b"\xff" + self.source,
        }
        for label, content in mutations.items():
            with self.subTest(label=label):
                self.asm.write_bytes(content if isinstance(content, bytes) else content.encode())
                self.run_patch()

    def test_second_invocation_fails_without_write(self):
        self.run_patch(True)
        self.run_patch()

    def test_invalid_or_unavailable_goroot(self):
        for value in ["", "relative", str(self.root / "missing")]:
            with self.subTest(goroot=value):
                self.info["GOROOT"] = value
                self.run_patch()

    def test_missing_assembly(self):
        self.asm.rename(self.asm.with_suffix('.saved'))
        env = dict(os.environ, PATH=str(self.root), FIXTURE_GO_ENV=json.dumps(self.info))
        result = subprocess.run([sys.executable, str(PATCH)], env=env, capture_output=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(self.asm.exists())

    def test_unavailable_go(self):
        self.fake.unlink()
        self.run_patch()

    def test_go_env_failure(self):
        self.fake.write_text('#!/bin/sh\nexit 7\n')
        self.run_patch()

    def test_invalid_go_env_json(self):
        self.fake.write_text('#!/bin/sh\nprintf bad-json\n')
        self.run_patch()

    def test_unsupported_version(self):
        self.info["GOVERSION"] = "go1.28.0"
        self.assertIn("unsupported Go version", self.run_patch().stderr)

    def test_atomic_replace_failure_preserves_original(self):
        with mock.patch.object(patch.subprocess, "check_output", return_value=json.dumps(self.info)), \
                mock.patch.object(patch.os, "replace", side_effect=OSError("injected rename failure")):
            with self.assertRaisesRegex(SystemExit, "injected rename failure"):
                patch.main()
        self.assertEqual(self.asm.read_bytes(), self.source)
        self.assertEqual(list(self.asm.parent.glob('.a7-*')), [])

    def test_atomic_flush_failure_preserves_original(self):
        with mock.patch.object(patch.subprocess, "check_output", return_value=json.dumps(self.info)), \
                mock.patch.object(patch.os, "fsync", side_effect=OSError("injected flush failure")):
            with self.assertRaisesRegex(SystemExit, "injected flush failure"):
                patch.main()
        self.assertEqual(self.asm.read_bytes(), self.source)
        self.assertEqual(list(self.asm.parent.glob('.a7-*')), [])

    def test_symlink_source_is_rejected(self):
        saved = self.asm.with_suffix('.saved')
        self.asm.rename(saved)
        self.asm.symlink_to(saved)
        self.run_patch()
        self.assertEqual(saved.read_bytes(), self.source)

    def test_zero_guard_and_exact_loop(self):
        self.assertEqual(patch.shape(patch.REPLACEMENT), (
            patch.START, "MOVWU cycles+0(FP), R0", "CBZ R0, done", "again:",
            "YIELD", "SUBW $1, R0", "CBNZ R0, again", "done:", "RET"))


if __name__ == "__main__":
    unittest.main()
