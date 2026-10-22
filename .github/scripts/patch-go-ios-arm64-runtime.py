#!/usr/bin/env python3
"""Patch only the reviewed Go 1.27.3 ARM64 procyield body, atomically."""

import hashlib
import json
import os
from pathlib import Path
import stat
import subprocess
import tempfile

SUPPORTED_VERSION = "go1.27.3"
SUPPORTED_SOURCE_SHA256 = "47f9313d4f956b4159dcf8b292bf340a12adb6b58ee9a94360fe948045e2c5db"
START = "TEXT runtime·procyieldAsm(SB),NOSPLIT,$0-0"
END = "\n// Save state of caller into g->sched,"
# Complete executable source shape from go1.27.3/src/runtime/asm_arm64.s.
# Comments and horizontal whitespace carry no assembly semantics. All other
# lines (including directives, labels and unexpected instructions) must match.
expected_shape = (
    START,
    "MOVWU cycles+0(FP), R0", "CBZ R0, done", "ISB $15",
    "CMP $18, R0", "BLS done", "SUB $18, R0, R0",
    "MRS CNTFRQ_EL0, R1", "ADD R0>>4, R0, R0", "MUL R1, R0, R0",
    "LSR $30, R0, R0", "CBZ R0, done", "MRS CNTVCT_EL0, R2",
    "delay:", "ISB $15", "MRS CNTVCT_EL0, R1", "SUB R2, R1, R1",
    "CMP R0, R1", "BCC delay", "done:", "RET",
)
REPLACEMENT = """TEXT runtime·procyieldAsm(SB),NOSPLIT,$0-0
\tMOVWU\tcycles+0(FP), R0
\tCBZ\tR0, done
again:
\tYIELD
\tSUBW\t$1, R0
\tCBNZ\tR0, again
done:
\tRET
"""


def shape(body):
    return tuple(" ".join(code.split()) for line in body.splitlines()
                 if (code := line.split("//", 1)[0].strip()))


def patch_source(source):
    if source.count(START) != 1 or source.count(END) != 1:
        raise ValueError("expected exactly one function start and end marker")
    start, end = source.index(START), source.index(END)
    if start and source[start - 1] != "\n":
        raise ValueError("function start is not a line boundary")
    if end <= start or shape(source[start:end]) != expected_shape:
        raise ValueError("procyieldAsm complete executable source shape changed")
    # No other bytes are rewritten; the zero case must return without underflow.
    return source[:start] + REPLACEMENT + source[end:]


def main():
    temporary = None
    try:
        info = json.loads(subprocess.check_output(
            ["go", "env", "-json", "GOROOT", "GOVERSION"], text=True))
        if not isinstance(info, dict):
            raise ValueError("go env returned invalid toolchain metadata")
        if info.get("GOVERSION") != SUPPORTED_VERSION:
            raise ValueError(f"unsupported Go version {info.get('GOVERSION')!r}; expected {SUPPORTED_VERSION}")
        root = Path(info.get("GOROOT", ""))
        if not root.is_absolute() or not root.is_dir():
            raise ValueError("go env GOROOT must resolve an existing absolute directory")
        path = root / "src/runtime/asm_arm64.s"
        if path.is_symlink() or not path.is_file():
            raise ValueError(f"Go runtime source missing or symlinked: {path}")
        original = path.read_bytes()
        source_sha256 = hashlib.sha256(original).hexdigest()
        if source_sha256 != SUPPORTED_SOURCE_SHA256:
            raise ValueError(f"unknown Go 1.27.3 runtime source SHA-256: {source_sha256}")
        patched = patch_source(original.decode("utf-8")).encode("utf-8")
        # Same-directory temporary file + rename avoids a truncated runtime if a
        # write fails. Verification completes before creating the temporary file.
        with tempfile.NamedTemporaryFile(dir=path.parent, prefix=".a7-", delete=False) as stream:
            temporary = Path(stream.name)
            os.fchmod(stream.fileno(), stat.S_IMODE(path.stat().st_mode))
            stream.write(patched)
            stream.flush()
            os.fsync(stream.fileno())
        if path.read_bytes() != original:
            raise ValueError("runtime source changed during patch validation")
        os.replace(temporary, path)
        temporary = None
        print(json.dumps({"status": "PASS", "go": info["GOVERSION"], "path": str(path),
                          "source_sha256": source_sha256,
                          "patched_sha256": hashlib.sha256(patched).hexdigest(),
                          "replacement": "zero-guarded YIELD loop; no CNTFRQ_EL0/CNTVCT_EL0 reads"}))
    except (OSError, ValueError, KeyError, subprocess.CalledProcessError) as exc:
        raise SystemExit(f"ERROR: {exc}; refusing to patch unknown Go runtime") from exc
    finally:
        if temporary is not None:
            temporary.unlink(missing_ok=True)


if __name__ == "__main__":
    main()
