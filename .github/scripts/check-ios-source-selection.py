#!/usr/bin/env python3
"""Validate real Go file selection for the Agent/Hub and dependency graph."""

import argparse
import json
import os
from pathlib import Path
import subprocess

MODULE = "github.com/henrygd/beszel"
GOPSUTIL = "github.com/shirou/gopsutil/v4"


def decode_packages(text):
    decoder, packages = json.JSONDecoder(), {}
    while text.strip():
        item, end = decoder.raw_decode(text.lstrip())
        text = text.lstrip()[end:]
        if item.get("Error") or item.get("DepsErrors"):
            raise ValueError(f"Go dependency selection error: {item['ImportPath']}")
        if item['ImportPath'] in packages:
            raise ValueError("duplicate Go package")
        packages[item['ImportPath']] = item
    return packages


def selected(package):
    return set(package.get('GoFiles', []) + package.get('CgoFiles', []) + package.get('SFiles', []))


def validate(packages, target):
    def require(path, names):
        package = packages.get(path)
        if package is None:
            raise ValueError(f"{target}: missing package {path}")
        files = selected(package)
        if not set(names) <= files:
            raise ValueError(f"{target}: {path}: missing {set(names) - files}")
        return package, files

    require(MODULE + '/internal/cmd/agent', ['agent.go'])
    hub, _ = require(MODULE + '/internal/cmd/hub', ['hub.go'])
    if hub.get('Name') != 'main' or packages[MODULE + '/internal/cmd/agent'].get('Name') != 'main':
        raise ValueError('entry point must be package main')
    site, _ = require(MODULE + '/internal/site', ['embed.go'])
    embeds = site.get('EmbedFiles', [])
    if ('dist/index.html' not in embeds or
            not any(f.startswith('dist/assets/') and f.endswith('.js') for f in embeds) or
            not any(f.startswith('dist/assets/') and f.endswith('.css') for f in embeds)):
        raise ValueError('missing real production frontend embed assets')

    _, agent = require(MODULE + '/agent', ['system.go', 'cpu.go', 'memory.go', 'disk.go', 'network.go'])
    # Require exactly one implementation from each known platform family.
    pairs = {
        'system_platform': ['system_platform_ios.go', 'system_platform_other.go'],
        'gpu_darwin': ['gpu_darwin.go', 'gpu_darwin_unsupported.go'],
        'cpu': ['cpu_linux.go', 'cpu_unsupported.go'],
        'memory': ['memory_linux.go', 'memory_unsupported.go'],
        'network_counters': ['network_counters_linux.go', 'network_counters_unsupported.go'],
        'systemd': ['systemd.go', 'systemd_nonlinux.go'],
    }
    for family, alternatives in pairs.items():
        expected = ('system_platform_ios.go' if target == 'ios' else 'system_platform_other.go') if family == 'system_platform' else (
            'gpu_darwin.go' if target == 'darwin' else 'gpu_darwin_unsupported.go') if family == 'gpu_darwin' else (
            alternatives[0] if target == 'linux' else alternatives[1])
        if agent.intersection(alternatives) != {expected}:
            raise ValueError(f'{target}: invalid {family} implementations: {agent.intersection(alternatives)}')
    package_updates = {'package_updates_ios.go', 'package_updates_otheros.go'}
    expected_package_updates = 'package_updates_ios.go' if target == 'ios' else 'package_updates_otheros.go'
    if agent.intersection(package_updates) != {expected_package_updates}:
        raise ValueError(f'{target}: invalid package-update adapter selection: {agent.intersection(package_updates)}')
    if target == 'ios':
        require(MODULE + '/agent', ['memory_platform_ios.go', 'ios_memory.go', 'ios_metadata.go', 'ios_disk.go'])
    # Optional collectors must select genuine no-op platform adapters on iOS.
    if target == 'ios':
        require(MODULE + '/agent', ['sensors_ios.go', 'emmc_stub.go', 'mdraid_stub.go', 'fans_other.go'])
        if agent.intersection({'sensors_default.go', 'sensors_windows.go', 'sensors_freebsd.go'}):
            raise ValueError('iOS selects a foreign temperature collector')
    else:
        require(MODULE + '/agent', ['sensors_default.go'])
    _, zfs = require(MODULE + '/agent/zfs', ['zfs_ios.go' if target == 'ios' else 'zfs_linux.go' if target == 'linux' else 'zfs_nonlinux.go'])
    if target == 'ios' and 'zfs_nonlinux.go' in zfs:
        raise ValueError('iOS selects utility-based ZFS capability detection')
    require(MODULE + '/agent/btrfs', ['btrfs_linux.go' if target == 'linux' else 'btrfs_unsupported.go'])
    require(MODULE + '/agent/battery', ['ios_ioreg.go'])
    for subsystem in ['battery', 'wifi']:
        package, files = require(MODULE + '/agent/' + subsystem, [subsystem + '.go'])
        expected = f'{subsystem}_{target}.go'
        implementations = {f for f in files if f.startswith(subsystem + '_')}
        if implementations != {expected}:
            raise ValueError(f'{target}: invalid {subsystem} implementations: {implementations}')
    for path, package in packages.items():
        files = selected(package)
        if target != 'ios' and any(f.endswith('_ios.go') for f in files):
            raise ValueError(f'{target}: iOS source leaked into {path}')
        if target != 'linux' and any(f.endswith('_linux.go') for f in files):
            raise ValueError(f'{target}: Linux source leaked into {path}')
    if target == 'ios' and 'gpu_darwin_test.go' in packages[MODULE + '/agent'].get('TestGoFiles', []):
        raise ValueError('iOS selects unsupported macOS GPU tests')
    # Darwin constraints apply to iOS too. These dependency selections are
    # intentional; their syscall/command runtime availability is P008/P010.
    for subsystem in ['cpu', 'mem', 'disk', 'net', 'host', 'load', 'sensors']:
        suffix = 'darwin' if target in ['ios', 'darwin'] else 'linux'
        expected = f'{subsystem}_{suffix}.go'
        if subsystem == 'sensors' and suffix == 'darwin':
            expected = 'sensors_darwin_arm64.go'
        require(GOPSUTIL + '/' + subsystem, [expected])
    return {path: {key: package.get(key, []) for key in
                   ['GoFiles', 'CgoFiles', 'SFiles', 'TestGoFiles', 'IgnoredGoFiles', 'EmbedFiles']}
            for path, package in sorted(packages.items())
            if path.startswith(MODULE + '/agent') or path.startswith(GOPSUTIL + '/') or
            path in [MODULE + '/internal/site', MODULE + '/internal/cmd/agent', MODULE + '/internal/cmd/hub', 'runtime/cgo']}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path)
    args = parser.parse_args()
    result = {}
    for target in ['ios', 'darwin', 'linux']:
        env = dict(os.environ, GOOS=target, GOARCH='arm64', CGO_ENABLED='1', GOARM64='v8.0', GOFLAGS='')
        output = subprocess.check_output(['go', 'list', '-mod=readonly', '-deps', '-json',
                                         './internal/cmd/agent', './internal/cmd/hub'], env=env, text=True)
        result[target] = validate(decode_packages(output), target)
        print(f'PASS {target}/arm64 CGO=1: Agent/Hub, platform families, gopsutil, frontend embed')
    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(result, indent=2) + '\n')


if __name__ == '__main__':
    main()
