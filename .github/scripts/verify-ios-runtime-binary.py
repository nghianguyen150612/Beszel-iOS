#!/usr/bin/env python3
"""Check the complete symbol-level A7 replacement in an unstripped build."""

import argparse
import json
from pathlib import Path
import re
import subprocess

# Go 1.27.2 ARM64 assembler output, including branch displacements. This checks
# more than absence of named counter registers: every instruction is known.
EXPECTED = ['b9400be0', 'b4000080', 'd503203f', '51000400', 'b5ffffc0', 'd65f03c0']


def validate(disassembly):
    headings = re.findall(r'^TEXT ([^ ]+)\(SB\)', disassembly, re.MULTILINE)
    if len(headings) != 1 or headings[0] not in ['runtime.procyieldAsm', 'runtime.procyieldAsm.abi0']:
        raise ValueError('expected exactly one procyieldAsm symbol; unstripped validation build required')
    words = re.findall(r'^\s+\S+:\d+\s+0x[0-9a-f]+\s+([0-9a-f]{8})\s+', disassembly, re.MULTILINE)
    if words[:6] != EXPECTED or any(word != '00000000' for word in words[6:]) or len(words) not in [6, 8]:
        raise ValueError('runtime.procyieldAsm instructions differ from verified zero-guarded YIELD loop')
    return {'symbol': headings[0], 'instructions': words, 'counter_access': False, 'status': 'PASS'}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('binary', type=Path)
    parser.add_argument('--output', type=Path)
    args = parser.parse_args()
    try:
        assembly = subprocess.check_output(['go', 'tool', 'objdump', '-s',
                                            r'^runtime\.procyieldAsm(\.abi0)?$', str(args.binary)], text=True)
        result = validate(assembly)
    except (OSError, ValueError, subprocess.CalledProcessError) as exc:
        raise SystemExit(f'ERROR: {exc}') from exc
    print(assembly)
    print(json.dumps(result, indent=2))
    if args.output:
        args.output.write_text(json.dumps(result, indent=2) + '\n')


if __name__ == '__main__':main()
