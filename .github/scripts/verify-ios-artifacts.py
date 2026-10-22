#!/usr/bin/env python3
"""Verify the exact iOS payload, Mach-O metadata and genuine frontend embedding."""

import argparse
import hashlib
import json
from pathlib import Path
import re
import struct
import subprocess

NAMES = ('beszel-agent-ios-arm64', 'beszel-hub-ios-arm64')


def macho(data):
    if len(data) < 32:
        raise ValueError('empty or truncated Mach-O header')
    magic, cpu, subtype, filetype, count, size, flags, reserved = struct.unpack_from('<8I', data)
    if magic != 0xFEEDFACF or cpu != 0x0100000C or subtype & 0xFFFFFF != 0 or filetype != 2:
        raise ValueError('expected thin ARM64 (not ARM64e) Mach-O executable')
    end = 32 + size
    if end > len(data) or count > size // 8:
        raise ValueError('truncated load commands')
    cursor, versions, entries, executable = 32, [], [], []
    for _ in range(count):
        if cursor + 8 > end:
            raise ValueError('missing load command')
        command, length = struct.unpack_from('<2I', data, cursor)
        if length < 8 or length % 8 or cursor + length > end:
            raise ValueError('invalid load command size')
        if command == 0x32:  # LC_BUILD_VERSION
            if length < 24:
                raise ValueError('truncated LC_BUILD_VERSION')
            platform, minimum, sdk, tools = struct.unpack_from('<4I', data, cursor + 8)
            if length != 24 + tools * 8 or platform != 2 or minimum != 12 << 16:
                raise ValueError('LC_BUILD_VERSION must be iOS platform 2, minos 12.0')
            versions.append({'command': 'LC_BUILD_VERSION', 'platform': platform, 'minos': '12.0', 'sdk': sdk})
        elif command in (0x24, 0x25, 0x2F, 0x30):
            if command != 0x25 or length != 16 or struct.unpack_from('<I', data, cursor + 8)[0] != 12 << 16:
                raise ValueError('legacy version command must be LC_VERSION_MIN_IPHONEOS 12.0')
            versions.append({'command': 'LC_VERSION_MIN_IPHONEOS', 'platform': 2, 'minos': '12.0'})
        elif command == 0x19:  # LC_SEGMENT_64
            if length < 72:
                raise ValueError('truncated segment')
            vmaddr, vmsize, fileoff, filesize, maxprot, initprot, sections, segflags = struct.unpack_from('<4Q4I', data, cursor + 24)
            if length != 72 + sections * 80 or fileoff + filesize > len(data):
                raise ValueError('invalid segment bounds')
            if initprot & 4:
                executable.append((fileoff, fileoff + filesize, vmaddr, vmaddr + vmsize))
        elif command == 0x80000028:  # LC_MAIN
            if length != 24:
                raise ValueError('invalid LC_MAIN')
            entries.append(('file', struct.unpack_from('<Q', data, cursor + 8)[0]))
        elif command == 5:  # LC_UNIXTHREAD / ARM_THREAD_STATE64
            if length != 288 or struct.unpack_from('<2I', data, cursor + 8) != (6, 68):
                raise ValueError('unsupported UNIXTHREAD entry state')
            entries.append(('vm', struct.unpack_from('<Q', data, cursor + 272)[0]))
        cursor += length
    if cursor != end or len(versions) != 1:
        raise ValueError('expected exactly one iOS deployment command')
    if len(entries) != 1:
        raise ValueError('expected exactly one executable entry point')
    kind, entry = entries[0]
    if not any((lo <= entry < hi) for flo, fhi, vlo, vhi in executable
               for lo, hi in [(flo, fhi) if kind == 'file' else (vlo, vhi)]):
        raise ValueError('entry point outside executable segment')
    return {'cpu': 'arm64', 'filetype': 'MH_EXECUTE', **versions[0], 'entry_kind': kind, 'entry': entry}


def frontend_assets(directory):
    html = directory / 'index.html'
    if not html.is_file() or not html.stat().st_size:
        raise ValueError('missing real frontend index.html')
    references = re.findall(r'(?:src|href)="((?:\./|/)?assets/[^"?#]+)"', html.read_text())
    if not any(p.endswith('.js') for p in references) or not any(p.endswith('.css') for p in references):
        raise ValueError('frontend index must reference production JS and CSS')
    for reference in references:
        path = directory / reference.lstrip('/')
        if not path.resolve().is_relative_to(directory.resolve()) or not path.is_file() or not path.stat().st_size:
            raise ValueError(f'missing or empty frontend asset: {reference}')
    return [p for p in sorted(directory.rglob('*')) if p.is_file()]


def build_info(text, name, expected_commit=None):
    version = re.search(r'^.+: (go[0-9.]+)$', text, re.MULTILINE)
    path = re.search(r'^\s+path\s+(\S+)$', text, re.MULTILINE)
    settings = dict(re.findall(r'^\s+build\s+([\w.]+)=(.*)$', text, re.MULTILINE))
    command = 'agent' if name == NAMES[0] else 'hub'
    if not version or version[1] != 'go1.27.3' or not path or path[1] != 'github.com/henrygd/beszel/internal/cmd/' + command:
        raise ValueError(f'{name}: wrong Go version or Agent/Hub entry package')
    for key, value in {'GOOS': 'ios', 'GOARCH': 'arm64', 'GOARM64': 'v8.0', 'CGO_ENABLED': '1'}.items():
        if settings.get(key) != value:
            raise ValueError(f'{name}: wrong Go build setting {key}')
    if expected_commit and (settings.get('vcs.revision') != expected_commit or settings.get('vcs.modified') != 'false'):
        raise ValueError(f'{name}: Go VCS metadata must match the clean tested feature commit')
    return {'go': version[1], 'entry_package': path[1], 'settings': settings}


def verify(directory, frontend=None, inspect_go=False, expected_commit=None):
    if {p.name for p in directory.iterdir()} != {*NAMES, 'SHA256SUMS'}:
        raise ValueError('payload must contain exactly Agent, Hub and SHA256SUMS')
    manifest = {}
    for line in (directory / 'SHA256SUMS').read_text().splitlines():
        match = re.fullmatch(r'([0-9a-f]{64}) [ *](beszel-(?:agent|hub)-ios-arm64)', line)
        if not match or match[2] in manifest:
            raise ValueError('invalid or duplicate manifest entry (bare payload filenames required)')
        manifest[match[2]] = match[1]
    if set(manifest) != set(NAMES):
        raise ValueError('manifest must include both binaries exactly once')
    result = {}
    for name in NAMES:
        path = directory / name
        if path.is_symlink() or not path.is_file():
            raise ValueError(f'invalid payload file: {name}')
        data = path.read_bytes()
        digest = hashlib.sha256(data).hexdigest()
        if manifest[name] != digest:
            raise ValueError(f'checksum mismatch: {name}')
        result[name] = {**macho(data), 'sha256': digest, 'bytes': len(data)}
        if inspect_go:
            metadata = subprocess.check_output(['go', 'version', '-m', str(path)], text=True)
            result[name]['build_info'] = build_info(metadata, name, expected_commit)
        if name == NAMES[1] and frontend is not None:
            assets = frontend_assets(frontend)
            for asset in assets:
                if asset.read_bytes() not in data:
                    raise ValueError(f'frontend asset is not embedded in Hub: {asset.relative_to(frontend)}')
            result[name]['embedded_asset_count'] = len(assets)
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('directory', type=Path)
    parser.add_argument('--frontend', type=Path)
    parser.add_argument('--output', type=Path)
    parser.add_argument('--expected-commit')
    args = parser.parse_args()
    try:
        result = verify(args.directory, args.frontend, inspect_go=True, expected_commit=args.expected_commit)
    except (ValueError, OSError, struct.error, subprocess.CalledProcessError) as exc:
        raise SystemExit(f'ERROR: {exc}') from exc
    report = json.dumps(result, indent=2) + '\n'
    print(report, end='')
    if args.output:
        args.output.write_text(report)


if __name__ == '__main__':
    main()
