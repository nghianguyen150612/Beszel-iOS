#!/usr/bin/env python3
"""Validate pinned declarations without confusing exports with a verified ABI."""
import argparse
import hashlib
import json
from pathlib import Path
import re

PINNED = {
    'WiFiManager.h': 'b4549c720b5766307a0888ba0c736ad2d940334ad101a101c996e127c31b803d',
    'WiFiDeviceClient.h': 'ee33e3699db2bb564b9627429baa3af8402f7804f94942938e6f9524a0bdcac6',
    'WiFiNetwork.h': '68c963636c1529258761ec81b7e6a64dbb02ed4f63d420cf58329683b3d61835',
    'MobileWiFi.tbd': 'f7ac468998f286a1e0bc850a16c1f8c12027b6032f7485c41ac4660bc679efdf',
}
FUNCTIONS = {
    'WiFiManagerClientCreate': ('WiFiManager.h', 'opaque manager', 'owned by historical Create convention'),
    'WiFiManagerClientCopyDevices': ('WiFiManager.h', 'CFArrayRef', 'owned; array elements borrowed while array lives'),
    'WiFiDeviceClientCopyProperty': ('WiFiDeviceClient.h', 'CFPropertyListRef', 'owned; dictionary children borrowed while dictionary lives'),
    'WiFiDeviceClientCopyCurrentNetwork': ('WiFiDeviceClient.h', 'opaque network', 'owned by historical Copy convention'),
    'WiFiDeviceClientGetInterfaceName': ('WiFiDeviceClient.h', 'CFStringRef', 'borrowed; device owner must remain alive'),
    'WiFiNetworkGetSSID': ('WiFiNetwork.h', 'CFStringRef', 'borrowed; network owner must remain alive'),
}


def declarations(directory):
    data = {}
    for name, expected in PINNED.items():
        raw = (directory / name).read_bytes()
        if hashlib.sha256(raw).hexdigest() != expected:
            raise ValueError('unexpected pinned source bytes: ' + name)
        data[name] = raw.decode()
    result = []
    for function, (header, kind, ownership) in FUNCTIONS.items():
        found = re.findall(r'([^;{}\n]+\b' + re.escape(function) + r'\s*\([^;]*?\)\s*;)', data[header])
        if len(found) != 1:
            raise ValueError('missing or duplicate historical declaration: ' + function)
        if '_' + function not in data['MobileWiFi.tbd']:
            raise ValueError('missing iOS 12.4 SDK export: ' + function)
        result.append({'name': function, 'declaration': ' '.join(found[0].split()),
                       'declared_result_kind': kind, 'historical_ownership': ownership,
                       'historical_declaration_status': 'VERIFIED',
                       'ios_12_4_sdk_export_status': 'VERIFIED',
                       'ios_12_5_7_export_status': 'UNKNOWN',
                       'ios_12_5_7_typed_abi_status': 'UNKNOWN',
                       'ownership_on_reference_firmware_status': 'STRONGLY_INFERRED',
                       'calling_convention': 'historical C declaration; arm64 pointer/int ABI inferred, not verified by symbol name',
                       'null_policy': 'NULL means unavailable; never dereference; release only non-NULL objects with established ownership',
                       'release_policy': ownership, 'header': header})
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--sources', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    args.output.write_text(json.dumps({'schema_version': 1, 'symbols': declarations(args.sources),
                                      'pinned_source_hashes': PINNED}, indent=2) + '\n')


if __name__ == '__main__':
    main()
