#!/usr/bin/env python3
"""Verify the development IPA, exact-source provenance and Mach-O build metadata."""
import hashlib
import pathlib
import plistlib
import struct
import sys
import zipfile

ipa = pathlib.Path(sys.argv[1])
if len(sys.argv) > 2:
    source = ipa.with_name("SOURCE.sha").read_text().strip()
    assert source == sys.argv[2], f"unexpected source SHA: {source}"
    print(f"Source commit: {source}")
digest, filename = ipa.with_name("SHA256SUMS").read_text().split()
assert filename == ipa.name, "checksum filename mismatch"
with ipa.open("rb") as ipa_file:
    assert hashlib.file_digest(ipa_file, "sha256").hexdigest() == digest, "IPA checksum mismatch"

with zipfile.ZipFile(ipa) as archive:
    assert archive.testzip() is None, "corrupt IPA ZIP"
    names = archive.namelist()
    root = "Payload/BeszelStockAgent.app/"
    assert len(names) == len(set(names)), "duplicate ZIP entries"
    assert all(name == "Payload/" or name.startswith(root) for name in names), "unexpected IPA payload"
    assert not any({"..", "."} & set(name.split("/")) for name in names), "invalid IPA paths"
    info = plistlib.loads(archive.read(root + "Info.plist"))
    expected = {
        "MinimumOSVersion": "16.0",
        "CFBundleExecutable": "BeszelStockAgent",
        "CFBundleIdentifier": "dev.nghianguyen.beszel.stockagent",
        "CFBundlePackageType": "APPL",
        "CFBundleShortVersionString": "0.1.0",
        "CFBundleVersion": "1",
        "CFBundleSupportedPlatforms": ["iPhoneOS"],
        "UIDeviceFamily": [1],
    }
    for key, value in expected.items():
        assert info[key] == value, f"unexpected {key}: {info[key]}"
    assert not info.get("UIBackgroundModes"), "unexpected background modes"
    assert root + "_CodeSignature/CodeResources" in names, "signature resources missing"
    assert root + "embedded.mobileprovision" not in names, "unexpected provisioning profile"
    if len(sys.argv) > 2:
        privacy = plistlib.loads(archive.read(root + "PrivacyInfo.xcprivacy"))
        assert privacy["NSPrivacyAccessedAPITypes"] == [{
            "NSPrivacyAccessedAPIType": "NSPrivacyAccessedAPICategoryDiskSpace",
            "NSPrivacyAccessedAPITypeReasons": ["85F4.1"],
        }, {
            "NSPrivacyAccessedAPIType": "NSPrivacyAccessedAPICategoryUserDefaults",
            "NSPrivacyAccessedAPITypeReasons": ["CA92.1"],
        }], "wrong stock privacy reasons"
    data = archive.read(root + "BeszelStockAgent")

magic, cpu, subtype, filetype, count, size, flags, reserved = struct.unpack_from("<8I", data)
assert magic == 0xFEEDFACF and cpu == 0x0100000C and filetype == 2, "not arm64 Mach-O executable"
assert 32 + size <= len(data), "invalid load command size"
offset, builds, signature = 32, [], None
for _ in range(count):
    command, length = struct.unpack_from("<2I", data, offset)
    assert length >= 8 and offset + length <= 32 + size, "invalid load command"
    if command == 0x32:  # LC_BUILD_VERSION
        assert length >= 24
        platform, minimum, sdk, tools = struct.unpack_from("<4I", data, offset + 8)
        builds.append((platform, minimum, sdk))
    if command == 0x1D:  # LC_CODE_SIGNATURE
        assert length >= 16 and signature is None
        start, sig_size = struct.unpack_from("<2I", data, offset + 8)
        assert 0 < sig_size <= len(data) - start, "invalid embedded signature"
        signature = data[start:start + sig_size]
    offset += length
assert offset == 32 + size
assert len(builds) == 1 and builds[0][:2] == (2, 16 << 16), f"wrong build target: {builds}"
assert b"\x00_BeszelAgentStart\x00" in data and b"\x00_BeszelAgentUpdateMetrics\x00" in data, "Go bridge exports missing"
assert b"go1.27.1\x00" in data, "Go toolchain marker missing"
assert signature is not None, "unsigned IPA"
magic, blob_size, blobs = struct.unpack_from(">3I", signature)
assert magic == 0xFADE0CC0 and 12 + blobs * 8 <= blob_size <= len(signature), "malformed signature superblob"
ad_hoc = False
for i in range(blobs):
    slot, start = struct.unpack_from(">2I", signature, 12 + i * 8)
    assert start + 8 <= blob_size
    magic, length = struct.unpack_from(">2I", signature, start)
    assert 8 <= length <= blob_size - start
    assert slot not in (5, 7), "unexpected embedded entitlements"
    if slot == 0:  # Primary CodeDirectory
        assert magic == 0xFADE0C02 and length >= 16
        flags = struct.unpack_from(">I", signature, start + 12)[0]
        ad_hoc = bool(flags & 0x2)  # CS_ADHOC
assert ad_hoc, "missing ad-hoc CodeDirectory flag"
sdk = builds[0][2]
print(f"PASS: {info['CFBundleIdentifier']} {info['CFBundleShortVersionString']} ({info['CFBundleVersion']})")
print(f"iPhoneOS arm64, minimum iOS 16.0, SDK {sdk >> 16}.{sdk >> 8 & 255}.{sdk & 255}")
print(f"Go 1.27.1 bridge, ad-hoc CodeDirectory, no provisioning, IPA SHA-256 {digest}")
