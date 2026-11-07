#!/usr/bin/env python3
"""Hermetic archive identity fixtures; no real build, signature or Apple calls."""
import contextlib
import importlib.util
import io
import os
from pathlib import Path
import plistlib
import struct
import subprocess
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('archive_check', Path(__file__).with_name('check-ios-archive.py'))
checker = importlib.util.module_from_spec(spec)
spec.loader.exec_module(checker)


def binary(platform=2, cpu=0x0100000C, subtype=0):
    """Create platform stamps only, deliberately not a runnable/signed executable."""
    command = struct.pack('<6I', 0x32, 24, platform, 13 << 16, 26 << 16, 0)
    return struct.pack('<8I', 0xFEEDFACF, cpu, subtype, 2, 1, len(command), 0, 0) + command


def binary_plist(objects, aliases=None):
    """Build compact object tables, optionally pointing indices into other objects."""
    data = b'bplist00'
    offsets = []
    for value in objects:
        offsets.append(len(data))
        data += value
    for index, target in (aliases or {}).items():
        offsets[index] = target(offsets)
    table = len(data)
    width = 1 if table < 256 else 2 if table < 65536 else 4
    return data + b''.join(value.to_bytes(width, 'big') for value in offsets) + struct.pack('>6xBBQQQ', width, 1, len(objects), 0, table)


class Archives(unittest.TestCase):
    """Exercise local metadata and binary-stamp boundaries with generated fixtures."""

    def setUp(self):
        """Build an owned synthetic archive tree with known public identity fields."""
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.archive = Path(self.temp.name) / 'Fixture.xcarchive'
        self.app = self.archive / 'Products/Applications/Runner.app'
        self.app.mkdir(parents=True)
        self.properties = {'ApplicationPath': 'Applications/Runner.app', 'CFBundleIdentifier': checker.BUNDLE,
                           'CFBundleShortVersionString': '1.0.0', 'CFBundleVersion': '7'}
        self.info = {'CFBundleIdentifier': checker.BUNDLE, 'CFBundleShortVersionString': '1.0.0',
                     'CFBundleVersion': '7', 'CFBundleExecutable': 'Runner', 'CFBundlePackageType': 'APPL',
                     'DTPlatformName': 'iphoneos', 'CFBundleSupportedPlatforms': ['iPhoneOS']}
        self.write_metadata()
        (self.app / 'Runner').write_bytes(binary())

    def write_metadata(self, fmt=plistlib.FMT_BINARY):
        """Write only fixture metadata in the requested supported plist encoding."""
        (self.archive / 'Info.plist').write_bytes(plistlib.dumps({'ApplicationProperties': self.properties}, fmt=fmt))
        (self.app / 'Info.plist').write_bytes(plistlib.dumps(self.info, fmt=fmt))

    def check(self):
        """Check this fixed candidate against caller-selected public version/build."""
        checker.check_archive(self.archive, '1.0.0', '7')

    def test_valid_binary_and_xml_metadata(self):
        """Both plist formats pass only metadata identity, never release approval."""
        self.check()
        self.write_metadata(plistlib.FMT_XML)
        self.check()

    def test_bundle_version_build_mismatches(self):
        """Refuse an unrelated app or mismatched release identity before reuse."""
        for key, value in (('CFBundleIdentifier', 'other.app'), ('CFBundleShortVersionString', '2.0.0'),
                           ('CFBundleVersion', '8')):
            original = dict(self.info)
            with self.subTest(field=key):
                self.info[key] = value
                self.write_metadata()
                with self.assertRaises(checker.Refused):
                    self.check()
            self.info = original

    def test_outer_archive_identity_disagrees(self):
        """Present outer identity fields must match the app and requested values."""
        self.properties['CFBundleVersion'] = '999'
        self.write_metadata()
        with self.assertRaises(checker.Refused):
            self.check()

    def test_simulator_binary_with_device_plist(self):
        """A forged device plist cannot disguise the simulator platform stamp."""
        (self.app / 'Runner').write_bytes(binary(platform=7))
        with self.assertRaises(checker.Refused):
            self.check()

    def test_wrong_cpu_stamp(self):
        """An x86_64 executable is refused even with device platform metadata."""
        (self.app / 'Runner').write_bytes(binary(cpu=0x01000007))
        with self.assertRaises(checker.Refused):
            self.check()

    def test_invalid_or_unsupported_binary_headers(self):
        """Thin-profile and load-command bounds fail before unbounded reads."""
        valid = binary()
        cases = {'truncated': valid[:31], 'fat': b'\xca\xfe\xba\xbe' + valid[4:],
                 'subtype': binary(subtype=2), 'oversized commands': valid[:20] + struct.pack('<I', checker.COMMAND_LIMIT + 8) + valid[24:],
                 'truncated command': valid[:-1], 'missing stamp': valid[:32] + struct.pack('<2I', 0x99, 24) + valid[40:],
                 'invalid command length': valid[:36] + struct.pack('<I', 16) + valid[40:]}
        for name, data in cases.items():
            with self.subTest(case=name):
                (self.app / 'Runner').write_bytes(data)
                with self.assertRaises(checker.Refused):
                    self.check()

    def test_duplicate_and_legacy_platform_stamps(self):
        """Contradictory or ambiguous platform declarations are unsupported."""
        valid = binary()
        duplicate = struct.pack('<8I', 0xFEEDFACF, 0x0100000C, 0, 2, 2, 48, 0, 0) + valid[32:] * 2
        legacy = struct.pack('<8I', 0xFEEDFACF, 0x0100000C, 0, 2, 2, 40, 0, 0) + valid[32:] + struct.pack('<4I', 0x25, 16, 13 << 16, 26 << 16)
        for data in (duplicate, legacy):
            with self.subTest(size=len(data)):
                (self.app / 'Runner').write_bytes(data)
                with self.assertRaises(checker.Refused):
                    self.check()

    def test_simulator_metadata_refused(self):
        """Device binary bytes cannot compensate for simulator app metadata."""
        self.info['DTPlatformName'] = 'iphonesimulator'
        self.write_metadata()
        with self.assertRaises(checker.Refused):
            self.check()

    def test_application_path_escape_refused(self):
        """Only a single canonical Applications child may select the app."""
        for value in ('../Runner.app', '/Applications/Runner.app', 'Applications/../Runner.app',
                      'Applications/Runner.app/Other.app', 'Applications/%2e%2e.app'):
            with self.subTest(path=value):
                self.properties['ApplicationPath'] = value
                self.write_metadata()
                with self.assertRaises(checker.Refused):
                    self.check()

    def test_multiple_applications_refused(self):
        """No arbitrary choice is made between multiple apps in the archive."""
        (self.app.parent / 'Other.app').mkdir()
        with self.assertRaises(checker.Refused):
            self.check()

    def test_symlink_components_and_files_refused(self):
        """Directory descriptors and no-follow file opens prevent linked escapes."""
        executable = self.app / 'Runner'
        saved = self.app / 'RetainedBinary'
        executable.rename(saved)
        executable.symlink_to(saved)
        with self.assertRaises((checker.Refused, OSError)):
            self.check()
        executable.unlink()
        saved.rename(executable)
        linked = Path(self.temp.name) / 'Linked.xcarchive'
        linked.symlink_to(self.archive, target_is_directory=True)
        with self.assertRaises((checker.Refused, OSError)):
            checker.check_archive(linked, '1.0.0', '7')
        products = self.archive / 'Products'
        retained = self.archive / 'RetainedProducts'
        products.rename(retained)
        products.symlink_to(retained, target_is_directory=True)
        with self.assertRaises((checker.Refused, OSError)):
            self.check()

    def test_executable_path_escape_refused(self):
        """The executable is a single regular child, never a traversal path."""
        self.info['CFBundleExecutable'] = '../Info.plist'
        self.write_metadata()
        with self.assertRaises(checker.Refused):
            self.check()

    def test_sparse_oversized_executable_refused(self):
        """The file-size bound applies without reading an oversized sparse file."""
        with (self.app / 'Runner').open('r+b') as handle:
            handle.truncate((1 << 30) + 1)
        with self.assertRaises(checker.Refused):
            self.check()

    def test_changed_metadata_refused(self):
        """Mutation after parsing is observed before identity success is returned."""
        original = checker.device_binary
        def mutate(fd):
            """Simulate an unrelated writer changing an already-read metadata file."""
            original(fd)
            self.info['added-during-check'] = 'a definite size change'
            self.write_metadata()
        with patch.object(checker, 'device_binary', side_effect=mutate):
            with self.assertRaisesRegex(checker.Refused, 'changed while'):
                self.check()

    def test_refusal_closes_owned_descriptors(self):
        """Early metadata refusal closes all file descriptors opened by the checker."""
        self.info['CFBundleIdentifier'] = 'other.app'
        self.write_metadata()
        original = checker.os.open
        opened = []
        def observe(*args, **kwargs):
            """Record descriptors created only during this fixture check."""
            fd = original(*args, **kwargs)
            opened.append(fd)
            return fd
        with patch.object(checker.os, 'open', side_effect=observe):
            with self.assertRaises(checker.Refused):
                self.check()
        for fd in opened:
            with self.assertRaises(OSError):
                os.fstat(fd)

    def test_cli_positive_scope_and_private_error_redaction(self):
        """CLI success is identity-only and invalid private paths are not rendered."""
        script = Path(__file__).with_name('check-ios-archive.py').resolve()
        args = ['/usr/bin/python3', '-I', str(script), '--archive', str(self.archive), '--version', '1.0.0', '--build', '7']
        result = subprocess.run(args, capture_output=True, text=True, timeout=20)
        self.assertEqual(result.returncode, 0)
        self.assertIn('PASS: archive identity', result.stdout)
        self.assertIn('UNVERIFIED: signature', result.stdout)
        args[args.index('--archive') + 1] = str(Path(self.temp.name) / 'PRIVATE_ARTIFACT_PATH.xcarchive')
        result = subprocess.run(args, capture_output=True, text=True, timeout=20)
        self.assertEqual(result.returncode, 1)
        self.assertNotIn('PRIVATE_ARTIFACT_PATH', result.stdout + result.stderr)
        self.assertNotIn('Traceback', result.stdout + result.stderr)


class Plists(unittest.TestCase):
    """Reject ambiguous and resource-exhausting metadata before identity use."""

    def test_duplicate_xml_keys_refused(self):
        """A later matching value must not hide an earlier conflicting key."""
        data = b'<plist version="1.0"><dict><key>key</key><string>one</string><key>key</key><string>two</string></dict></plist>'
        with self.assertRaises(checker.Refused):
            checker.decode_plist(data)

    def test_duplicate_binary_keys_refused(self):
        """Binary-plist dictionary pairs receive the same duplicate-key check."""
        objects = [b'\xd2\x01\x01\x02\x03', b'\x53key', b'\x511', b'\x512']
        data = b'bplist00'
        offsets = []
        for value in objects:
            offsets.append(len(data))
            data += value
        table = len(data)
        data += bytes(offsets) + struct.pack('>6xBBQQQ', 1, 1, 4, 0, table)
        with self.assertRaises(checker.Refused):
            checker.decode_plist(data)

    def test_entities_depth_and_object_bounds_refused(self):
        """Refuse entity declarations, excessive XML depth and huge binary tables."""
        inputs = [b'<!DOCTYPE plist [<!ENTITY x "value">]><plist><dict/></plist>',
                  b'<plist>' + b'<array>' * 65 + b'<string>x</string>' + b'</array>' * 65 + b'</plist>',
                  b'bplist00' + struct.pack('>6xBBQQQ', 1, 1, 50001, 0, 8), b'x' * (checker.PLIST_LIMIT + 1)]
        for data in inputs:
            with self.subTest(size=len(data)), self.assertRaises(checker.Refused):
                checker.decode_plist(data)

    def test_binary_alias_overlap_cycle_and_depth_refused_before_decode(self):
        """Compact adversarial graphs cannot multiply decoder allocation or recurse."""
        aliased = binary_plist([b'\xd2\x01\x02\x03\x04', b'\x51a', b'\x51b', b'\xa1\x05', b'', b'\x51v'],
                               {4: lambda offsets: offsets[3]})
        overlapping = binary_plist([b'\xd2\x01\x02\x03\x04', b'\x51a', b'\x51b', b'\x43\x51xy', b''],
                                   {4: lambda offsets: offsets[3] + 1})
        cycle = binary_plist([b'\xd1\x01\x00', b'\x51a'])
        chain = [b'\xd1\x01\x02', b'\x51a']
        chain += [bytes((0xA1, index + 1)) for index in range(2, 68)]
        chain.append(b'\x51v')
        deep = binary_plist(chain)
        many = binary_plist([b'\xd1\x01\x02', b'\x51a',
                             b'\xaf\x12' + struct.pack('>I', 100001) + b'\x03' * 100001, b'\x51v'])
        for name, data in [('offset alias', aliased), ('overlap', overlapping), ('cycle', cycle), ('depth', deep), ('references', many)]:
            with self.subTest(case=name), patch.object(checker.plistlib, 'loads') as decode:
                with self.assertRaises(checker.Refused):
                    checker.decode_plist(data)
                decode.assert_not_called()

    def test_binary_shared_object_reference_is_preserved(self):
        """Legitimate DAG references reuse one object index without offset aliasing."""
        data = binary_plist([b'\xd2\x01\x02\x03\x03', b'\x51a', b'\x51b', b'\xa1\x04', b'\x51v'])
        value = checker.decode_plist(data)
        self.assertEqual(value, {'a': ['v'], 'b': ['v']})
        self.assertIs(value['a'], value['b'])

    def test_expected_version_and_build_bounds(self):
        """Bad expected values fail before any artifact file is opened."""
        for version, build in (('1.0', '7'), ('1.0.0', '0'), ('1.0.0', '07'), ('01.0.0', '7')):
            with self.subTest(version=version, build=build), patch.object(checker.os, 'open') as opened:
                with self.assertRaises(checker.Refused):
                    checker.check_archive('missing.xcarchive', version, build)
                opened.assert_not_called()


if __name__ == '__main__':
    unittest.main(verbosity=2)
