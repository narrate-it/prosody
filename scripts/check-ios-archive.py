#!/usr/bin/env python3
"""Read-only archive identity checks; never validate signing or approve release."""
import argparse
import contextlib
import io
import os
from pathlib import Path
import plistlib
import re
import signal
import stat
import struct
import sys
import xml.etree.ElementTree as ET

BUNDLE = 'run.sire.prosody'
PLIST_LIMIT = 1 << 20
COMMAND_LIMIT = 1 << 20
APPLE_DTD = b'<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">'


class Refused(Exception):
    """Fixed diagnostics that do not render private artifact paths or metadata."""


class UniqueDict(dict):
    """Reject duplicate plist keys instead of silently selecting a value."""

    def __init__(self, pairs=(), **kwargs):
        """Route initial pairs through the same duplicate-key check."""
        super().__init__()
        source = pairs.items() if hasattr(pairs, 'items') else pairs
        for key, value in source:
            self[key] = value
        for key, value in kwargs.items():
            self[key] = value

    def __setitem__(self, key, value):
        """Admit only unique string keys; never include their values in errors."""
        if not isinstance(key, str) or key in self:
            raise Refused('Ambiguous archive metadata keys')
        super().__setitem__(key, value)


def validate_binary_plist(data):
    """Bound allocations before plistlib follows binary object/reference tables."""
    if len(data) < 40:
        raise Refused('Invalid binary archive metadata')
    offset_size, ref_size, count, top, table = struct.unpack('>6xBBQQQ', data[-32:])
    if (offset_size not in (1, 2, 4, 8) or ref_size not in (1, 2, 4, 8)
            or not 1 <= count <= 50000 or top >= count or table < 8
            or table + count * offset_size != len(data) - 32):
        raise Refused('Binary archive metadata structure exceeds bounds')
    offsets = [int.from_bytes(data[table + index * offset_size:table + (index + 1) * offset_size], 'big')
               for index in range(count)]
    if len(set(offsets)) != count or any(not 8 <= offset < table for offset in offsets):
        raise Refused('Binary archive object offsets are invalid or aliased')
    edges = []
    extents = []
    references = 0
    for offset in offsets:
        marker = data[offset]
        kind, small = marker >> 4, marker & 15
        cursor = offset + 1
        refs = []
        if kind in (4, 5, 6, 10, 13):
            length = small
            if small == 15:
                if cursor >= table or data[cursor] >> 4 != 1 or data[cursor] & 15 > 3:
                    raise Refused('Invalid binary archive length object')
                width = 1 << (data[cursor] & 15)
                cursor += 1
                if cursor + width > table:
                    raise Refused('Truncated binary archive length')
                length = int.from_bytes(data[cursor:cursor + width], 'big')
                cursor += width
            multiplier = 2 if kind in (6, 13) else 1
            elements = length * multiplier
            if kind in (10, 13):
                references += elements
                if references > 100000:
                    raise Refused('Binary archive reference allocation exceeds bound')
                end = cursor + elements * ref_size
                if end > table:
                    raise Refused('Truncated binary archive references')
                refs = [int.from_bytes(data[cursor + index * ref_size:cursor + (index + 1) * ref_size], 'big')
                        for index in range(elements)]
                if any(value >= count for value in refs):
                    raise Refused('Binary archive reference is outside the object table')
            else:
                end = cursor + elements
        elif kind == 0 and small in (0, 8, 9):
            end = cursor
        elif kind == 1 and small <= 4:
            end = cursor + (1 << small)
        elif kind == 2 and small in (2, 3):
            end = cursor + (1 << small)
        elif kind == 3 and small == 3:
            end = cursor + 8
        elif kind == 8 and small <= 7:
            end = cursor + small + 1
        else:
            raise Refused('Unsupported binary archive object type')
        if end > table:
            raise Refused('Binary archive object exceeds its data region')
        extents.append((offset, end))
        edges.append(refs)
    previous = 8
    for start, end in sorted(extents):
        if start < previous:
            raise Refused('Binary archive object extents overlap')
        previous = end
    state = [0] * count
    heights = [0] * count

    def visit(index, depth):
        """Reject cycles/deep DAG paths with bounded memoized graph traversal."""
        if depth > 64 or state[index] == 1:
            raise Refused('Binary archive graph is cyclic or exceeds depth bound')
        if state[index] == 2:
            if depth + heights[index] - 1 > 64:
                raise Refused('Binary archive graph exceeds depth bound')
            return heights[index]
        state[index] = 1
        height = 1
        for child in edges[index]:
            height = max(height, 1 + visit(child, depth + 1))
        state[index] = 2
        heights[index] = height
        return height

    for index in range(count):
        visit(index, 1)


def decode_plist(data):
    """Bound plist structure and reject entities before using the standard decoder."""
    if not data or len(data) > PLIST_LIMIT:
        raise Refused('Archive metadata exceeds its byte bound')
    try:
        if data.startswith(b'bplist00'):
            validate_binary_plist(data)
        else:
            data.decode('utf-8', errors='strict')
            if b'<!ENTITY' in data.upper() or b'<!DOCTYPE' in data.replace(APPLE_DTD, b'').upper():
                raise Refused('Archive metadata entity or DTD refused')
            depth = nodes = 0
            for event, _ in ET.iterparse(io.BytesIO(data), events=('start', 'end')):
                if event == 'start':
                    depth += 1
                    nodes += 1
                    if depth > 64 or nodes > 50000:
                        raise Refused('Archive metadata structure exceeds bounds')
                else:
                    depth -= 1
        value = plistlib.loads(data, dict_type=UniqueDict)
        if not isinstance(value, dict):
            raise Refused('Archive metadata is not a dictionary')
        return value
    except (ValueError, UnicodeError, OverflowError, RecursionError, ET.ParseError, plistlib.InvalidFileException):
        raise Refused('Invalid archive metadata') from None


def snapshot(fd):
    """Observe inode and mutation metadata, not an immutable artifact capability."""
    value = os.fstat(fd)
    return (value.st_dev, value.st_ino, value.st_mode, value.st_size,
            value.st_mtime_ns, value.st_ctime_ns)


def child_directory(stack, parent, name, observed):
    """Open a descendant directory without following an archive symlink."""
    fd = os.open(name, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=parent)
    stack.callback(os.close, fd)
    observed.append((fd, snapshot(fd)))
    return fd


def regular_file(stack, parent, name, limit, observed):
    """Keep a bounded regular file open for post-read mutation checks."""
    fd = os.open(name, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=parent)
    stack.callback(os.close, fd)
    value = os.fstat(fd)
    if not stat.S_ISREG(value.st_mode) or not 0 < value.st_size <= limit:
        raise Refused('Artifact file type or size is outside the supported profile')
    observed.append((fd, snapshot(fd)))
    return fd


def metadata(stack, parent, observed):
    """Read one bounded Info.plist from its anchored directory descriptor."""
    fd = regular_file(stack, parent, 'Info.plist', PLIST_LIMIT, observed)
    data = bytearray()
    while len(data) <= PLIST_LIMIT:
        chunk = os.read(fd, min(65536, PLIST_LIMIT + 1 - len(data)))
        if not chunk:
            break
        data.extend(chunk)
    return decode_plist(bytes(data))


def device_binary(fd):
    """Check modern thin ARM64/iOS load-command stamps, not executable validity."""
    header = os.read(fd, 32)
    if len(header) != 32:
        raise Refused('Truncated executable header')
    magic, cpu, subtype, kind, count, size, _, reserved = struct.unpack('<8I', header)
    if (magic != 0xFEEDFACF or cpu != 0x0100000C or subtype != 0 or kind != 2
            or reserved != 0 or not 1 <= count <= 512 or not 8 <= size <= COMMAND_LIMIT):
        raise Refused('Only modern thin device ARM64 executables are supported')
    data = bytearray()
    while len(data) < size:
        chunk = os.read(fd, min(65536, size - len(data)))
        if not chunk:
            raise Refused('Truncated executable load commands')
        data.extend(chunk)
    offset = builds = 0
    for _ in range(count):
        if offset + 8 > len(data):
            raise Refused('Truncated executable load command')
        command, length = struct.unpack_from('<2I', data, offset)
        if length < 8 or length % 8 or offset + length > len(data):
            raise Refused('Invalid executable load-command bounds')
        if command in (0x24, 0x25, 0x2F, 0x30):
            raise Refused('Legacy or conflicting platform stamps are unsupported')
        if command == 0x32:
            if length < 24:
                raise Refused('Truncated build-version command')
            platform, minimum, sdk, tools = struct.unpack_from('<4I', data, offset + 8)
            if platform != 2 or not 0 < minimum <= sdk or tools > 32 or length != 24 + tools * 8:
                raise Refused('Executable is not stamped for the supported iOS device profile')
            builds += 1
        offset += length
    if builds != 1 or offset != len(data):
        raise Refused('Missing or ambiguous executable platform stamp')


def check_archive(path, version, build):
    """Match caller-selected release identity without asserting signing or provenance."""
    if (not isinstance(version, str) or not re.fullmatch(r'(?:0|[1-9][0-9]{0,3})(?:\.(?:0|[1-9][0-9]{0,3})){2}', version)
            or not isinstance(build, str) or not re.fullmatch(r'[1-9][0-9]{0,17}', build)):
        raise Refused('Expected version or build is outside this checker profile')
    root = Path(path).absolute()
    if root.suffix != '.xcarchive':
        raise Refused('Expected an existing xcarchive directory')
    observed = []
    with contextlib.ExitStack() as stack:
        fd = os.open(root, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
        stack.callback(os.close, fd)
        observed.append((fd, snapshot(fd)))
        archive = metadata(stack, fd, observed)
        properties = archive.get('ApplicationProperties')
        if not isinstance(properties, dict):
            raise Refused('Archive application properties are missing')
        application = properties.get('ApplicationPath')
        if not isinstance(application, str) or not re.fullmatch(r'Applications/[A-Za-z0-9][A-Za-z0-9_.-]{0,63}\.app', application):
            raise Refused('Archive application path is outside the supported profile')
        products = child_directory(stack, fd, 'Products', observed)
        apps = child_directory(stack, products, 'Applications', observed)
        with os.scandir(apps) as entries:
            names = []
            for entry in entries:
                names.append(entry.name)
                if len(names) > 64:
                    raise Refused('Archive application directory exceeds its entry bound')
        name = application.split('/')[1]
        if [value for value in names if value.endswith('.app')] != [name]:
            raise Refused('Archive has missing or ambiguous applications')
        app = child_directory(stack, apps, name, observed)
        info = metadata(stack, app, observed)
        expected = {'CFBundleIdentifier': BUNDLE, 'CFBundleShortVersionString': version, 'CFBundleVersion': build}
        if any(info.get(key) != value for key, value in expected.items()):
            raise Refused('App identity does not match the expected release')
        if any(key in properties and properties[key] != value for key, value in expected.items()):
            raise Refused('Archive and app identity metadata disagree')
        if info.get('DTPlatformName') != 'iphoneos' or info.get('CFBundleSupportedPlatforms') != ['iPhoneOS'] or info.get('CFBundlePackageType') != 'APPL':
            raise Refused('App metadata is not for the supported iOS device profile')
        executable = info.get('CFBundleExecutable')
        if (not isinstance(executable, str) or not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9_.-]{0,63}', executable)
                or executable in ('.', '..')):
            raise Refused('Invalid app executable name')
        binary = regular_file(stack, app, executable, 1 << 30, observed)
        device_binary(binary)
        if any(snapshot(opened) != initial for opened, initial in observed):
            raise Refused('Archive changed while its identity was checked')
        current = os.stat(root, follow_symlinks=False)
        if (current.st_dev, current.st_ino) != observed[0][1][:2]:
            raise Refused('Archive path changed while checking')


class SafeParser(argparse.ArgumentParser):
    """Avoid echoing private artifact paths in argument errors."""

    def error(self, message):
        """Use a fixed diagnostic instead of rendering command-line values."""
        raise Refused('Invalid archive-check arguments')


def main():
    """Report metadata identity only; do not transform this into release authority."""
    def expired(signum, frame):
        """Bound this CLI process; local I/O custody still needs qualification."""
        raise Refused('Archive identity check deadline reached')
    previous = signal.signal(signal.SIGALRM, expired)
    signal.alarm(15)
    try:
        parser = SafeParser(description=__doc__)
        parser.add_argument('--archive', required=True)
        parser.add_argument('--version', required=True)
        parser.add_argument('--build', required=True)
        args = parser.parse_args()
        check_archive(args.archive, args.version, args.build)
    except Refused as error:
        print('BLOCKED: ' + str(error))
        return 1
    except Exception:
        print('BLOCKED: Archive could not be checked; private diagnostics suppressed')
        return 1
    finally:
        signal.alarm(0)
        signal.signal(signal.SIGALRM, previous)
    print('PASS: archive identity matches run.sire.prosody for the device ARM64 profile.')
    print('UNVERIFIED: signature, provisioning, provenance, SDK/upload rules and build-number availability.')
    return 0


if __name__ == '__main__':
    sys.exit(main())
