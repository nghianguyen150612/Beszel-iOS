#!/usr/bin/env python3

import subprocess
from pathlib import Path

goroot = Path(
    subprocess.check_output(
        ["go", "env", "GOROOT"],
        text=True,
    ).strip()
)

path = goroot / "src/runtime/asm_arm64.s"
src = path.read_text()

start_marker = "TEXT runtime·procyieldAsm(SB),NOSPLIT,$0-0"
end_marker = "\n// Save state of caller into g->sched,"

if start_marker not in src:
    raise SystemExit("procyieldAsm start marker not found")

start = src.index(start_marker)
end = src.index(end_marker, start)

old = src[start:end]

if "CNTVCT_EL0" not in old:
    raise SystemExit(
        "procyieldAsm does not contain CNTVCT_EL0; "
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

path.write_text(src[:start] + replacement + src[end:])

print(f"Patched {path}")
print("Replaced CNTVCT_EL0 procyield with legacy YIELD loop")
