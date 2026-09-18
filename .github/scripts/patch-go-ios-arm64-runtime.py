#!/usr/bin/env python3

import re
import subprocess
from pathlib import Path


def fail(message: str) -> None:
    raise SystemExit(f"ERROR: {message}")


try:
    goroot_text = subprocess.check_output(
        ["go", "env", "GOROOT"],
        text=True,
    ).strip()
except (OSError, subprocess.CalledProcessError) as exc:
    fail(f"could not resolve Go GOROOT: {exc}")

if not goroot_text:
    fail("go env GOROOT returned an empty path")

goroot = Path(goroot_text)
path = goroot / "src/runtime/asm_arm64.s"
if not path.is_file():
    fail(f"Go runtime source not found: {path}")

src = path.read_text(encoding="utf-8")

start_marker = "TEXT runtime·procyieldAsm(SB),NOSPLIT,$0-0"
end_marker = "\n// Save state of caller into g->sched,"

if src.count(start_marker) != 1:
    fail(
        "expected exactly one procyieldAsm start marker; "
        "refusing to patch unknown Go runtime"
    )

start = src.index(start_marker)
try:
    end = src.index(end_marker, start)
except ValueError:
    fail(
        "procyieldAsm end marker not found; "
        "refusing to patch unknown Go runtime"
    )

old = src[start:end]
normalized = re.sub(r"\s+", " ", old).strip()

# The A7 workaround is deliberately tied to the known Go ARM64 implementation.
# Checking only for the instruction name would risk rewriting a future
# implementation whose counter access has different semantics.
expected_shape = (
    "TEXT runtime·procyieldAsm(SB),NOSPLIT,$0-0",
    "MOVWU cycles+0(FP), R0",
    "CBZ R0, done",
    "MRS CNTFRQ_EL0, R1",
    "MRS CNTVCT_EL0, R2",
    "MRS CNTVCT_EL0, R1",
    "SUB R2, R1, R1",
    "BCC delay",
    "done: RET",
)
cursor = 0
for marker in expected_shape:
    position = normalized.find(marker, cursor)
    if position < 0:
        fail(
            f"procyieldAsm source shape changed (missing '{marker}'); "
            "refusing to patch unknown Go runtime"
        )
    cursor = position + len(marker)

if normalized.count("MRS CNTVCT_EL0") != 2:
    fail(
        "procyieldAsm source shape changed (expected two CNTVCT_EL0 reads); "
        "refusing to patch unknown Go runtime"
    )

replacement = """TEXT runtime·procyieldAsm(SB),NOSPLIT,$0-0
\tMOVWU\tcycles+0(FP), R0
again:
\tYIELD
\tSUBW\t$1, R0
\tCBNZ\tR0, again
\tRET
"""

patched = src[:start] + replacement + src[end:]
patched_body = patched[start : start + len(replacement)]
if "CNTVCT_EL0" in patched_body or "YIELD" not in patched_body:
    fail("runtime replacement verification failed")

path.write_text(patched, encoding="utf-8")

print(f"Patched {path}")
print("Replaced CNTVCT_EL0 procyield with legacy YIELD loop")
