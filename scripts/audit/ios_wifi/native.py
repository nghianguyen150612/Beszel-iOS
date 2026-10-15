#!/usr/bin/env python3
"""Build/validate a separate disabled diagnostic and public-CF native fixtures."""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[3]
SOURCE = ROOT / 'tools/ios-wifi-probe'
spec = importlib.util.spec_from_file_location('ios_artifacts', ROOT / '.github/scripts/verify-ios-artifacts.py')
artifacts = importlib.util.module_from_spec(spec)
spec.loader.exec_module(artifacts)


def checked(argv, **kwargs):
    return subprocess.check_output([str(x) for x in argv], text=True, **kwargs)


def loader_fixtures(output, flags):
    sanitizers = ['-fsanitize=address,undefined', '-fno-omit-frame-pointer']
    library_flags = ['-dynamiclib'] if sys.platform == 'darwin' else ['-shared', '-fPIC']
    libraries = {}
    for mode in ['missing', 'invalid', 'partial', 'all']:
        library = output / (mode + '.dylib')
        libraries[mode] = library
        if mode == 'invalid':
            library.write_text('not a loadable framework\n')
        elif mode != 'missing':
            extra = ['-DBESZEL_PARTIAL_SYMBOLS'] if mode == 'partial' else []
            subprocess.run([*flags, *library_flags, *extra, SOURCE/'fixture_symbols.c', '-o', library], check=True)
    fixture = output / 'loader-fixtures'
    libs = [] if sys.platform == 'darwin' else ['-ldl']
    subprocess.run([*flags, *sanitizers, SOURCE/'loader.c', SOURCE/'loader_test.c', *libs, '-o', fixture], check=True)
    outcomes = {}
    for mode, library in libraries.items():
        env = dict(os.environ, ASAN_OPTIONS='detect_leaks=0', BESZEL_WIFI_LOADER_TRACE=str(output/(mode+'.trace')))
        outcomes[mode] = checked([fixture, library, {'missing':'0','invalid':'0','partial':'1','all':'6'}[mode]], env=env).strip()
    return libraries, outcomes


def host(output, revision):
    sdk = checked(['xcrun', '--sdk', 'macosx', '--show-sdk-path']).strip()
    clang = checked(['xcrun', '--sdk', 'macosx', '--find', 'clang']).strip()
    flags = [clang, '-isysroot', sdk, '-g', '-Wall', '-Wextra']
    fixture = output / 'cf-fixtures'
    sanitizer_flags = ['-fsanitize=address,undefined', '-fno-omit-frame-pointer']
    subprocess.run([*flags, *sanitizer_flags, SOURCE/'core.c', SOURCE/'core_test.c', '-framework', 'CoreFoundation', '-o', fixture], check=True)
    result = checked([fixture], env=dict(os.environ, ASAN_OPTIONS='detect_leaks=0'))
    probe = output / 'beszel-ios-wifi-probe'
    subprocess.run([*flags, '-DBESZEL_WIFI_REVISION="'+revision+'"', SOURCE/'core.c', SOURCE/'loader.c', SOURCE/'probe.c', '-framework', 'CoreFoundation', '-o', probe], check=True)
    report = json.loads(checked([probe, '--snapshot-v1']))
    assert report == {'version': 1, 'source': 'mobilewifi-ios12', 'status': 'unavailable', 'reason': 'native-contract-unverified', 'devices': []}
    inventories = {}
    libraries, cleanup = loader_fixtures(output, flags)
    for mode, library in libraries.items():
        binary = output / ('inventory-'+mode)
        subprocess.run([*flags, '-DBESZEL_WIFI_FRAMEWORK_PATH="'+str(library)+'"', SOURCE/'core.c', SOURCE/'loader.c', SOURCE/'probe.c', '-framework', 'CoreFoundation', '-o', binary], check=True)
        inventory = json.loads(checked([binary, '--symbols-v1']))
        assert inventory['private_calls_executed'] is False and inventory['acquisition_enabled'] is False
        assert inventory['framework_available'] == (mode in ['partial','all'])
        assert sum(inventory['symbols'].values()) == {'missing':0, 'invalid':0, 'partial':1, 'all':6}[mode]
        inventories[mode] = inventory
    return {'public_cf_fixtures': 'PASS', 'sanitizers': 'address,undefined',
            'leak_sanitizer': 'disabled on macOS; custom CF allocator asserts all owned-container allocations released',
            'fixture_output': result.strip(), 'loader_inventory_fixtures': inventories, 'loader_cleanup_fixtures':cleanup,
            'snapshot': report, 'private_api_invocations': 0, 'helper': str(probe)}


def ios(output, revision):
    binary = output / 'beszel-ios-wifi-probe'
    command = [ROOT/'.github/scripts/ios-clang.sh', '-O2', '-g', '-march=armv8-a',
               '-DBESZEL_WIFI_REVISION="'+revision+'"', SOURCE/'core.c', SOURCE/'loader.c', SOURCE/'probe.c',
               '-framework', 'CoreFoundation', '-o', binary]
    subprocess.run(command, check=True)
    data = binary.read_bytes()
    info = artifacts.macho(data)
    marker = ('beszel-ios-wifi-probe revision='+revision+' protocol=1 activation=disabled').encode()
    if marker not in data:
        raise ValueError('diagnostic revision/activation attestation missing')
    linkage = checked(['otool', '-L', binary]) + checked(['nm', '-u', binary])
    if re.search(r'MobileWiFi|CoreWLAN|Apple80211|_WiFi(Manager|Device|Network)', linkage):
        raise ValueError('diagnostic has a hard private Wi-Fi import')
    (output/'linkage.txt').write_text(linkage)
    return {**info, 'sha256': hashlib.sha256(data).hexdigest(), 'bytes':len(data),
            'revision':revision, 'activation':'disabled', 'private_calls_executed':False,
            'compiler':checked(['xcrun','--sdk','iphoneos','clang','--version']).splitlines()[0],
            'sdk':checked(['xcrun','--sdk','iphoneos','--show-sdk-version']).strip(),
            'invocation':[str(x).replace(str(ROOT),'<source>') for x in command],
            'not_in_agent_hub_payload':True,
            'source_hashes':{n:hashlib.sha256((SOURCE/n).read_bytes()).hexdigest() for n in ['core.h','core.c','loader.h','loader.c','probe.c']}}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--host', action='store_true')
    parser.add_argument('--ios', action='store_true')
    parser.add_argument('--loader', action='store_true', help='portable production loader sanitizers only; no CF/native iOS execution')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--revision', required=True)
    args = parser.parse_args()
    if sum([args.host,args.ios,args.loader]) != 1 or not re.fullmatch('[0-9a-f]{40}',args.revision):
        raise ValueError('select one target and exact source revision')
    args.output.mkdir(parents=True, exist_ok=True)
    args.output = args.output.resolve()
    if args.loader:
        _, outcomes=loader_fixtures(args.output,[os.environ.get('CC','cc'),'-g','-Wall','-Wextra'])
        result={'loader_cleanup_fixtures':outcomes,'sanitizers':'address,undefined','private_calls_executed':False,'CF_executed':False}
    else:
        result = host(args.output,args.revision) if args.host else ios(args.output,args.revision)
    (args.output/'validation.json').write_text(json.dumps(result,indent=2)+'\n')
    print(json.dumps(result,indent=2))


if __name__=='__main__':
    main()
