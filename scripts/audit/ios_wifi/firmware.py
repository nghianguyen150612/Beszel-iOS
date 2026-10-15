#!/usr/bin/env python3
"""Inspect a checksum-pinned Apple restore image on a disposable macOS runner.

Only mounts a restore-image filesystem read-only. Never touches a physical device.
No binary from the image is executed; reports contain metadata and selected text.
"""
import argparse
import hashlib
import json
from pathlib import Path
import plistlib
import re
import shutil
import subprocess
import tempfile
import zipfile

VERSION = '12.5.7'
BUILD = '16H81'
MODEL = 'iPad4,4'
BOARD = 'j85ap'
IPSW_SHA256 = '780aa5b20fa9fb5770c18311663bd098dbbf8e89311675a19b4860303edc0cd9'
SYMBOLS = ['WiFiManagerClientCreate', 'WiFiManagerClientCopyDevices',
           'WiFiDeviceClientCopyProperty', 'WiFiDeviceClientCopyCurrentNetwork',
           'WiFiDeviceClientGetInterfaceName', 'WiFiNetworkGetSSID']


def identity(manifest):
    if (manifest.get('ProductVersion') != VERSION or
            manifest.get('ProductBuildVersion') != BUILD or
            MODEL not in manifest.get('SupportedProductTypes', [])):
        raise ValueError('restore manifest does not match reference firmware/model')
    matches = [x for x in manifest.get('BuildIdentities', [])
               if x.get('Info', {}).get('DeviceClass') == BOARD]
    paths = {x.get('Manifest', {}).get('OS', {}).get('Info', {}).get('Path') for x in matches}
    if not matches or len(paths) != 1 or None in paths:
        raise ValueError('missing or ambiguous reference-board root filesystem')
    path = paths.pop()
    if Path(path).is_absolute() or '..' in Path(path).parts:
        raise ValueError('invalid image path')
    return path


def inspect(image, mount, output):
    binary = mount / 'System/Library/PrivateFrameworks/MobileWiFi.framework/MobileWiFi'
    version = plistlib.loads((mount / 'System/Library/CoreServices/SystemVersion.plist').read_bytes())
    if version.get('ProductVersion') != VERSION or version.get('ProductBuildVersion') != BUILD:
        raise ValueError('mounted filesystem firmware mismatch')
    report = {'model': MODEL, 'board': BOARD, 'version': VERSION, 'build': BUILD,
              'ipsw_sha256': IPSW_SHA256, 'filesystem_version_verified': True,
              'private_calls_executed': False, 'device_accessed': False}
    if not binary.is_file():
        caches = [mount / 'System/Library/Caches/com.apple.dyld/dyld_shared_cache_arm64',
                  mount / 'System/Library/dyld/dyld_shared_cache_arm64']
        cache = next((p for p in caches if p.is_file()), None)
        if cache is None:
            report['binary_status'] = 'CACHE_NOT_FOUND'
            return report
        report['cache_path'] = str(cache.relative_to(mount))
        digest = hashlib.sha256()
        with cache.open('rb') as source:
            for chunk in iter(lambda: source.read(4 * 1024 * 1024), b''):
                digest.update(chunk)
        report['cache_sha256'] = digest.hexdigest()
        extractor = output.parent / 'cache-extract'
        clang = subprocess.check_output(['xcrun', '--sdk', 'macosx', '--find', 'clang'], text=True).strip()
        sdk = subprocess.check_output(['xcrun', '--sdk', 'macosx', '--show-sdk-path'], text=True).strip()
        subprocess.run([clang, '-isysroot', sdk, '-fblocks', str(Path(__file__).with_name('cache-extract.c')), '-o', str(extractor)], check=True)
        extraction = output.parent / 'extracted-cache'
        extraction.mkdir(exist_ok=True)
        try:
            subprocess.run([str(extractor), str(cache), str(extraction)], check=True, timeout=600)
        except (subprocess.CalledProcessError, subprocess.TimeoutExpired) as error:
            report['binary_status'] = 'CACHE_EXTRACTION_FAILED'
            report['extractor_error'] = type(error).__name__
            return report
        binary = extraction / 'System/Library/PrivateFrameworks/MobileWiFi.framework/MobileWiFi'
        if not binary.is_file():
            report['binary_status'] = 'MOBILEWIFI_NOT_EXTRACTED'
            return report
        report['binary_status'] = 'EXTRACTED_FROM_REFERENCE_CACHE'
    data = binary.read_bytes()
    report['binary_sha256'] = hashlib.sha256(data).hexdigest()
    report['binary_bytes'] = len(data)
    report['file'] = subprocess.check_output(['file', str(binary)], text=True).replace(str(mount), '<restore>')
    report['uuid'] = subprocess.check_output(['xcrun', 'dwarfdump', '--uuid', str(binary)], text=True).replace(str(mount), '<restore>')
    if 'arm64' not in report['file']:
        raise ValueError('expected arm64 firmware framework')
    # nm export presence verifies symbol availability only, never C prototypes.
    exports = subprocess.check_output(['nm', '-gjU', str(binary)], text=True).splitlines()
    report['exports'] = {name: '_' + name in exports for name in SYMBOLS}
    strings = subprocess.check_output(['strings', '-a', str(binary)], text=True, errors='replace').splitlines()
    report['selected_strings'] = [s for s in strings if s in ['RSSI', 'RSSI_CTL_AGR', 'com.apple.wifi.manager-access']]
    disassembly = subprocess.check_output(['otool', '-tvV', str(binary)], text=True)
    chunks = {}
    for name in SYMBOLS:
        marker = '_' + name + ':'
        lines = disassembly.splitlines()
        for index, line in enumerate(lines):
            if line.strip() == marker:
                chunk = []
                for following in lines[index + 1:]:
                    if following and not following[0].isspace() and following.endswith(':'):
                        break
                    chunk.append(following)
                    if len(chunk) >= 512:
                        break
                chunks[name] = '\n'.join(chunk)
                break
    report['disassembly_status'] = {name: 'EXTRACTED' if name in chunks else 'NO_SYMBOL_LABEL' for name in SYMBOLS}
    (output / 'selected-functions.txt').write_text('\n\n'.join(k + ':\n' + v for k, v in chunks.items()) + '\n')
    report['selected_functions_sha256'] = hashlib.sha256((output / 'selected-functions.txt').read_bytes()).hexdigest()
    # Daemon entitlement string is corroboration, not proof of minimum client rights.
    for candidate in ['usr/sbin/wifid', 'usr/libexec/wifid']:
        daemon = mount / candidate
        if daemon.is_file():
            text = subprocess.check_output(['strings', '-a', str(daemon)], text=True, errors='replace')
            report['wifid'] = {'path': candidate, 'sha256': hashlib.sha256(daemon.read_bytes()).hexdigest(),
                               'manager_access_string_present': 'com.apple.wifi.manager-access' in text}
            break
    return report


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('ipsw', type=Path)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    digest = hashlib.sha256()
    with args.ipsw.open('rb') as source:
        for block in iter(lambda: source.read(4 * 1024 * 1024), b''):
            digest.update(block)
    if digest.hexdigest() != IPSW_SHA256:
        raise ValueError('Apple restore image checksum mismatch')
    args.output.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory() as temporary:
        temporary = Path(temporary)
        with zipfile.ZipFile(args.ipsw) as archive:
            manifest = plistlib.loads(archive.read('BuildManifest.plist'))
            name = identity(manifest)
            image = temporary / 'filesystem.dmg'
            with archive.open(name) as source, image.open('wb') as dest:
                shutil.copyfileobj(source, dest, 4 * 1024 * 1024)
        mount = temporary / 'mount'
        mount.mkdir()
        attached = False
        try:
            subprocess.run(['hdiutil', 'attach', '-readonly', '-nobrowse', '-mountpoint', str(mount), str(image)], check=True)
            attached = True
            report = inspect(image, mount, args.output)
            report['filesystem_image'] = name
            (args.output / 'firmware-report.json').write_text(json.dumps(report, indent=2) + '\n')
            print(json.dumps(report, indent=2))
        finally:
            if attached:
                subprocess.run(['hdiutil', 'detach', str(mount)], check=True)


if __name__ == '__main__':
    main()
