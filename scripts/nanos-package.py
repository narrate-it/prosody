#!/usr/bin/env python3
"""Offline package-input inspection; never builds, boots or deploys an image."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import struct
import stat
import sys

ROOT = Path(__file__).resolve().parent.parent
MAX_BINARY_BYTES = 256 * 1024 * 1024
READ_CHUNK_BYTES = 1024 * 1024


def digest(data):
    """Return the SHA-256 hex digest of the supplied bytes."""
    return hashlib.sha256(data).hexdigest()


def inspect_binary(path):
    """Hash a bounded regular file and return its header, rejecting observed changes."""
    # O_NONBLOCK prevents a FIFO open from waiting for a writer. Reject all
    # nonregular inputs using this same descriptor before attempting a read.
    fd = os.open(path, os.O_RDONLY | os.O_NONBLOCK)
    try:
        before = os.fstat(fd)
        if not stat.S_ISREG(before.st_mode):
            raise ValueError('binary must be a regular file')
        if before.st_size > MAX_BINARY_BYTES:
            raise ValueError('binary exceeds 256 MiB inspection limit')
        hashed = hashlib.sha256()
        header = bytearray()
        total = 0
        while True:
            chunk = os.read(fd, min(READ_CHUNK_BYTES, MAX_BINARY_BYTES - total + 1))
            if not chunk:
                break
            total += len(chunk)
            if total > MAX_BINARY_BYTES or total > before.st_size:
                raise ValueError('binary grew during inspection or exceeds limit')
            hashed.update(chunk)
            if len(header) < 64:
                header.extend(chunk[:64 - len(header)])
        after = os.fstat(fd)
        identity = lambda value: (value.st_dev, value.st_ino, value.st_size,
                                  value.st_mtime_ns, value.st_ctime_ns)
        if total != before.st_size or identity(before) != identity(after):
            raise ValueError('binary changed during inspection')
        return bytes(header), {'sha256': hashed.hexdigest(), 'size': total}
    finally:
        os.close(fd)


def main():
    """Validate package inputs and print a static check or unqualified manifest."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--self-check', action='store_true')
    parser.add_argument('--binary', type=Path)
    parser.add_argument('--source-revision')
    parser.add_argument('--amos-revision')
    parser.add_argument('--ops-version')
    parser.add_argument('--nanos-version')
    args = parser.parse_args()
    contract_bytes = (ROOT / 'packaging/nanos/contract.json').read_bytes()
    config_bytes = (ROOT / 'packaging/nanos/ops-config.json').read_bytes()
    contract, config = json.loads(contract_bytes), json.loads(config_bytes)
    if contract['status'] != 'blocked-no-host-entrypoint' or config != {
        'Args': [], 'Env': {}, 'Files': [], 'RebootOnExit': False
    }:
        raise ValueError('template contract changed: review checker and runtime qualification')
    if (set(contract) != {'schemaVersion', 'status', 'target', 'opsConfig',
                         'runtimeInputs', 'qualificationRequired'}
            or contract['schemaVersion'] != 1 or contract['target'] != 'linux-amd64'
            or contract['opsConfig'] != 'packaging/nanos/ops-config.json'):
        raise ValueError('unsupported package contract')
    for field in ('runtimeInputs', 'qualificationRequired'):
        values = contract[field]
        if (not isinstance(values, list) or not values
                or any(not isinstance(value, str) or not re.fullmatch(r'[a-z][a-z0-9_]+', value)
                       for value in values) or len(values) != len(set(values))):
            raise ValueError('invalid contract input list: ' + field)
    if args.self_check:
        if any((args.binary, args.source_revision, args.amos_revision,
                args.ops_version, args.nanos_version)):
            raise ValueError('--self-check takes no package inputs')
        print('PASS: offline template consistency only; runtime/image qualification BLOCKED')
        return
    for field in ('binary', 'source_revision', 'amos_revision', 'ops_version', 'nanos_version'):
        if not getattr(args, field):
            raise ValueError('required input: --' + field.replace('_', '-'))
    for revision in (args.source_revision, args.amos_revision):
        if not re.fullmatch(r'[0-9a-f]{40}', revision):
            raise ValueError('source revisions require full lowercase Git commit IDs')
    # Accept release-shaped identifiers or complete Git object IDs, never arbitrary
    # branch/channel names. This syntax check does not verify remote provenance.
    release = r'v?(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)(?:-[0-9A-Za-z]+(?:[.-][0-9A-Za-z]+)*)?'
    for version in (args.ops_version, args.nanos_version):
        if len(version) > 128 or not (re.fullmatch(release, version)
                                      or re.fullmatch(r'[0-9a-f]{40}', version)):
            raise ValueError('explicit release identifier or full lowercase commit required; moving aliases prohibited')
    data, binary_manifest = inspect_binary(args.binary)
    # ELF64 little-endian x86-64 executable or PIE. This is structural inspection,
    # not evidence of Linux ABI, static linkage, provenance or Nanos compatibility.
    if (len(data) < 64 or data[:6] != b'\x7fELF\x02\x01'
            or struct.unpack_from('<H', data, 16)[0] not in (2, 3)
            or struct.unpack_from('<H', data, 18)[0] != 62):
        raise ValueError('expected ELF64 little-endian x86-64 executable input')
    manifest = {
        'schemaVersion': 1, 'status': 'inputs-inspected-runtime-unqualified',
        'sourceRevision': args.source_revision, 'amosRevision': args.amos_revision,
        'opsVersion': args.ops_version, 'nanosVersion': args.nanos_version,
        'binary': binary_manifest,
        'contractSha256': digest(contract_bytes), 'opsConfigSha256': digest(config_bytes),
    }
    print(json.dumps(manifest, sort_keys=True, indent=2))


if __name__ == '__main__':
    try:
        main()
    except (OSError, ValueError, KeyError) as error:
        print('nanos-package: ' + str(error), file=sys.stderr)
        sys.exit(1)
