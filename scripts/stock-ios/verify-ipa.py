#!/usr/bin/env python3
"""Verify exact development IPA payload, Mach-O target and checksum (host portable)."""
import hashlib
import pathlib
import plistlib
import struct
import sys
import zipfile

ipa = pathlib.Path(sys.argv[1])
checksum = ipa.with_name("SHA256SUMS").read_text().split()
assert checksum[1] == ipa.name, "checksum filename mismatch"
assert hashlib.sha256(ipa.read_bytes()).hexdigest() == checksum[0], "IPA checksum mismatch"
with zipfile.ZipFile(ipa) as archive:
    assert archive.testzip() is None, "corrupt ZIP"
    root = "Payload/BeszelStockAgent.app/"
    assert all(name == "Payload/" or name.startswith(root) for name in archive.namelist()), "unexpected IPA payload"
    info = plistlib.loads(archive.read(root + "Info.plist"))
    assert info["MinimumOSVersion"] == "16.0", "wrong plist minimum iOS"
    assert info["CFBundleExecutable"] == "BeszelStockAgent", "wrong executable"
    assert root + "_CodeSignature/CodeResources" in archive.namelist(), "signature resources missing"
    data = archive.read(root + "BeszelStockAgent")
magic, cpu, subtype, filetype, count, size, flags, reserved = struct.unpack_from("<8I", data)
assert magic == 0xFEEDFACF and cpu == 0x0100000C and filetype == 2, "not arm64 Mach-O executable"
offset, builds, signed = 32, [], False
for _ in range(count):
    command, length = struct.unpack_from("<2I", data, offset)
    assert length >= 8 and offset + length <= 32 + size, "invalid load command"
    if command == 0x32:
        platform, minimum, sdk, tools = struct.unpack_from("<4I", data, offset + 8)
        builds.append((platform, minimum))
    if command == 0x1D:
        start, length_signature = struct.unpack_from("<2I", data, offset + 8)
        signed = length_signature > 0 and start + length_signature <= len(data)
    offset += length
assert builds == [(2, 16 << 16)], f"wrong build platform/minimum: {builds}"
assert signed, "embedded signature missing"
print(f"PASS: iPhoneOS arm64, minimum iOS 16.0, IPA structure/signature presence/checksum {checksum[0]}")
