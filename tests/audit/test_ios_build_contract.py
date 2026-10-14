"""Negative metadata/payload/selection fixtures test the validators, not grep."""

import hashlib
import importlib.util
import json
from pathlib import Path
import struct
import os
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]


def module(name):
    spec = importlib.util.spec_from_file_location(name, ROOT / '.github/scripts' / (name + '.py'))
    result = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(result)
    return result


artifacts = module('verify-ios-artifacts')
selection = module('check-ios-source-selection')
runtime = module('verify-ios-runtime-binary')


def binary(cpu=0x0100000C, platform=2, minimum=12 << 16, legacy=False, entry=200):
    version = struct.pack('<4I', 0x25, 16, minimum, 17 << 16) if legacy else struct.pack('<6I', 0x32, 24, platform, minimum, 17 << 16, 0)
    segment = struct.pack('<2I16s4Q4I', 0x19, 72, b'__TEXT', 0x100000000, 4096, 0, 512, 5, 5, 0, 0)
    commands = segment + version + struct.pack('<2I2Q', 0x80000028, 24, entry, 0)
    return (struct.pack('<8I', 0xFEEDFACF, cpu, 0, 2, 3, len(commands), 0, 0) + commands).ljust(512, b'\x00')


class MachOTests(unittest.TestCase):
    def test_valid_ios_and_legacy(self):
        for legacy in [False, True]:
            self.assertEqual(artifacts.macho(binary(legacy=legacy))['minos'], '12.0')

    def test_wrong_arch_platform_deployment_and_entry(self):
        for options in [{'cpu': 0x01000007}, {'platform': 1}, {'platform': 7}, {'minimum': 13 << 16}, {'entry': 800}]:
            with self.subTest(options=options), self.assertRaises(ValueError):
                artifacts.macho(binary(**options))

    def test_missing_or_duplicate_version_commands(self):
        for command in [0x24, 0x2F, 0x30, 0x999]:
            data = bytearray(binary());struct.pack_into('<I', data, 104, command)
            with self.subTest(command=command), self.assertRaises(ValueError):
                artifacts.macho(data)
        data = bytearray(binary())
        # Replace LC_MAIN by a second complete LC_BUILD_VERSION.
        data[128:152] = data[104:128]
        with self.assertRaises(ValueError):artifacts.macho(data)

    def test_truncated_or_nonexecutable(self):
        for data in [b'', b'ELF', binary()[:120], binary()[:12] + struct.pack('<I', 6) + binary()[16:]]:
            with self.subTest(length=len(data)), self.assertRaises(ValueError):
                artifacts.macho(data)

    def test_invalid_command_size_and_count(self):
        for offset, value in [(36, 0), (16, 99), (20, 10000)]:
            data = bytearray(binary());struct.pack_into('<I', data, offset, value)
            with self.subTest(offset=offset), self.assertRaises(ValueError):
                artifacts.macho(data)

    def test_arm64e_excluded(self):
        data = bytearray(binary());struct.pack_into('<I', data, 8, 2)
        with self.assertRaises(ValueError):
            artifacts.macho(data)


class PayloadTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory();self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        for name in artifacts.NAMES:
            (self.root / name).write_bytes(binary())
        self.manifest = ''.join(hashlib.sha256(binary()).hexdigest() + '  ' + name + '\n' for name in artifacts.NAMES)
        (self.root / 'SHA256SUMS').write_text(self.manifest)

    def test_go_build_info_identifies_entry_package_and_target(self):
        info = 'binary: go1.27.1\n\tpath\tgithub.com/henrygd/beszel/internal/cmd/agent\n\tbuild\tGOOS=ios\n\tbuild\tGOARCH=arm64\n\tbuild\tCGO_ENABLED=1\n'
        artifacts.build_info(info, artifacts.NAMES[0])
        for invalid in [info.replace('/agent', '/hub'), info.replace('GOOS=ios','GOOS=darwin'),
                        info.replace('GOARCH=arm64','GOARCH=amd64'),info.replace('CGO_ENABLED=1','CGO_ENABLED=0'),
                        info.replace('go1.27.1','go1.28.0')]:
            with self.assertRaises(ValueError):artifacts.build_info(invalid, artifacts.NAMES[0])

    def test_valid_payload(self):
        self.assertEqual(set(artifacts.verify(self.root)), set(artifacts.NAMES))

    def test_missing_binaries_or_extra_packaging(self):
        for name in artifacts.NAMES:
            with self.subTest(name=name):
                path = self.root / name;path.unlink()
                with self.assertRaises(ValueError):artifacts.verify(self.root)
                path.write_bytes(binary())
        (self.root / 'extra').touch()
        with self.assertRaises(ValueError):artifacts.verify(self.root)

    def test_bad_checksums_absolute_paths_duplicates(self):
        for manifest in [self.manifest.replace('  beszel', '  /tmp/beszel'), self.manifest.replace(self.manifest[:64], '0' * 64), self.manifest + self.manifest, self.manifest.splitlines()[0]]:
            (self.root / 'SHA256SUMS').write_text(manifest)
            with self.subTest(manifest=manifest), self.assertRaises(ValueError):artifacts.verify(self.root)

    def test_frontend_must_be_real_and_embedded(self):
        with tempfile.TemporaryDirectory() as temp:
            frontend = Path(temp)
            with self.assertRaises(ValueError):artifacts.frontend_assets(frontend)
            (frontend / 'index.html').write_text('<html></html>')
            with self.assertRaises(ValueError):artifacts.frontend_assets(frontend)
            (frontend / 'index.html').write_text('<script src="/assets/app.js"></script><link href="/assets/app.css">')
            with self.assertRaises(ValueError):artifacts.frontend_assets(frontend)
            (frontend / 'assets').mkdir()
            for name in ['app.js', 'app.css']:(frontend / 'assets' / name).write_text('real production bytes')
            self.assertEqual(len(artifacts.frontend_assets(frontend)), 3)
            with self.assertRaisesRegex(ValueError, 'not embedded'):artifacts.verify(self.root, frontend)


def packages(target):
    # Deliberately small package graph with the same go-list JSON fields.
    result = {}
    def add(path, files, **extra):
        result[path] = {'ImportPath': path, 'Name': 'main' if '/cmd/' in path else 'package', 'GoFiles': files, **extra}
    prefix = selection.MODULE
    add(prefix + '/internal/cmd/agent', ['agent.go'])
    add(prefix + '/internal/cmd/hub', ['hub.go'])
    add(prefix + '/internal/site', ['embed.go'], EmbedFiles=['dist/index.html','dist/assets/app.js','dist/assets/app.css'])
    files = ['system.go','cpu.go','memory.go','disk.go','network.go',
             'system_platform_ios.go' if target == 'ios' else 'system_platform_other.go',
             'gpu_darwin.go' if target == 'darwin' else 'gpu_darwin_unsupported.go',
             'systemd.go' if target == 'linux' else 'systemd_nonlinux.go']
    files += [name + ('_linux.go' if target == 'linux' else '_unsupported.go') for name in ['cpu','memory','network_counters']]
    add(prefix + '/agent', files)
    for name in ['battery','wifi']:
        suffix = 'unsupported' if name == 'wifi' and target == 'ios' else target
        add(prefix + '/agent/' + name, [name + '.go', name + '_' + suffix + '.go'])
    for name in ['cpu','mem','disk','net','host','load','sensors']:
        suffix = 'linux' if target == 'linux' else 'darwin'
        if name == 'sensors' and suffix == 'darwin':suffix += '_arm64'
        add(selection.GOPSUTIL + '/' + name, [name + '_' + suffix + '.go'])
    return result


class SelectionTests(unittest.TestCase):
    def test_all_targets(self):
        for target in ['ios','darwin','linux']:selection.validate(packages(target), target)

    def test_unsupported_macos_collectors_and_duplicates(self):
        for path, filename in [('/agent','gpu_darwin.go'),('/agent','system_platform_other.go'),('/agent','cpu_linux.go'),('/agent/battery','battery_darwin.go'),('/agent/wifi','wifi_darwin.go')]:
            graph = packages('ios');graph[selection.MODULE + path]['GoFiles'].append(filename)
            with self.subTest(filename=filename), self.assertRaises(ValueError):selection.validate(graph, 'ios')

    def test_missing_dependencies_and_embed(self):
        for path in ['/internal/cmd/hub','/agent/battery']:
            graph = packages('ios');del graph[selection.MODULE + path]
            with self.assertRaises(ValueError):selection.validate(graph, 'ios')
        graph = packages('ios');graph[selection.MODULE + '/internal/site']['EmbedFiles'] = []
        with self.assertRaises(ValueError):selection.validate(graph, 'ios')

    def test_macos_test_excluded_on_ios(self):
        graph = packages('ios');graph[selection.MODULE + '/agent']['TestGoFiles'] = ['gpu_darwin_test.go']
        with self.assertRaises(ValueError):selection.validate(graph, 'ios')

    def test_ios_files_excluded_elsewhere(self):
        for target in ['darwin','linux']:
            graph = packages(target);graph[selection.MODULE + '/agent']['GoFiles'].append('unexpected_ios.go')
            with self.assertRaises(ValueError):selection.validate(graph, target)

    def test_go_list_errors_fail_closed(self):
        for fields in [{'Error': {'Err': 'bad'}}, {'DepsErrors': [{'Err':'bad'}]}]:
            with self.assertRaises(ValueError):selection.decode_packages(json.dumps({'ImportPath':'bad', **fields}))




class RuntimeBinaryTests(unittest.TestCase):
    def assembly(self, words):
        return 'TEXT runtime.procyieldAsm.abi0(SB) runtime/asm_arm64.s\n' + ''.join(
            f'  asm_arm64.s:700 0x{4096 + i * 4:x} {word} instruction\n' for i, word in enumerate(words))

    def test_exact_instructions(self):
        self.assertEqual(runtime.validate(self.assembly(runtime.EXPECTED + ['00000000'] * 2))['status'], 'PASS')

    def test_stripped_missing_extra_or_counter_instructions(self):
        for assembly in ['', self.assembly(runtime.EXPECTED[:-1]),
                         self.assembly(runtime.EXPECTED + ['d53be040']),
                         self.assembly(['d53be040'] + runtime.EXPECTED[1:]),
                         self.assembly(runtime.EXPECTED) * 2]:
            with self.assertRaises(ValueError):runtime.validate(assembly)


class CompilerWrapperTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory();self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.sdk = self.root / 'iPhoneOS17.5.sdk';self.sdk.mkdir()
        self.clang = self.root / 'clang'
        self.clang.write_text('#!/bin/sh\nprintf "%s\\n" "$@"\n')
        self.clang.chmod(0o755)
        self.xcrun = self.root / 'xcrun'
        self.xcrun.write_text('#!/bin/sh\n[ "$1 $2" = "--sdk iphoneos" ] || exit 99\ncase "$3" in --show-sdk-path) printf "%s\\n" "$FAKE_SDK";; --find) printf "%s\\n" "$FAKE_CLANG";; *) exit 98;; esac\n')
        self.xcrun.chmod(0o755)

    def invoke(self, args=()):
        return subprocess.run(['/bin/sh', str(ROOT / '.github/scripts/ios-clang.sh'), *args],
                              env=dict(os.environ, PATH=str(self.root), FAKE_SDK=str(self.sdk), FAKE_CLANG=str(self.clang)),
                              capture_output=True, text=True)

    def test_correct_device_flags(self):
        result = self.invoke(['-c', 'file.c']);self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.splitlines(), ['-arch','arm64','-isysroot',str(self.sdk),'-mios-version-min=12.0','-c','file.c'])

    def test_missing_components_and_host_flags(self):
        for args in [['-arch','x86_64'],['-target','arm64-apple-macos'],['-mmacosx-version-min=12.0'],['-mios-simulator-version-min=12.0'],['-isysroot','/macos']]:
            with self.subTest(args=args):self.assertNotEqual(self.invoke(args).returncode, 0)
        self.sdk.rmdir();self.assertNotEqual(self.invoke().returncode, 0)
        self.sdk = self.root / 'MacOSX.sdk';self.sdk.mkdir();self.assertNotEqual(self.invoke().returncode, 0)
        self.sdk = self.root / 'iPhoneOS17.5.sdk';self.sdk.mkdir();self.clang.unlink();self.assertNotEqual(self.invoke().returncode, 0)

if __name__ == '__main__':unittest.main()
