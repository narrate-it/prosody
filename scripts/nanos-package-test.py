#!/usr/bin/env python3
"""Hermetic CLI contract checks; synthetic headers are never runnable apps."""
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import struct
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

SCRIPT = Path(__file__).with_name('nanos-package.py')
SPEC = importlib.util.spec_from_file_location('nanos_package', SCRIPT)
PACKAGE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(PACKAGE)


class PackageInputs(unittest.TestCase):
    def run_cli(self, *args):
        return subprocess.run([sys.executable, str(SCRIPT), *args], capture_output=True, text=True, timeout=10)

    def test_static_template_and_missing_artifact_gate(self):
        self.assertEqual(self.run_cli('--self-check').returncode, 0)
        result = self.run_cli()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('required input: --binary', result.stderr)

    def test_reject_bad_artifact_and_moving_pins(self):
        with tempfile.TemporaryDirectory() as directory:
            artifact = Path(directory) / 'input'
            artifact.write_bytes(b'not an executable')
            args = ['--binary', str(artifact), '--source-revision', 'a' * 40,
                    '--amos-revision', 'b' * 40, '--ops-version', '1.2.3',
                    '--nanos-version', '1.2.3']
            self.assertIn('expected ELF64', self.run_cli(*args).stderr)
            args[-1] = 'latest'
            self.assertIn('moving aliases prohibited', self.run_cli(*args).stderr)

    def test_version_pin_grammar_for_both_tools(self):
        with tempfile.TemporaryDirectory() as directory:
            artifact = Path(directory) / 'header-fixture'
            header = bytearray(64)
            header[:6] = b'\x7fELF\x02\x01'
            struct.pack_into('<HH', header, 16, 2, 62)
            artifact.write_bytes(header)
            args = ['--binary', str(artifact), '--source-revision', 'a' * 40,
                    '--amos-revision', 'b' * 40, '--ops-version', '1.2.3',
                    '--nanos-version', '1.2.3']
            for index in (7, 9):
                for pin in ('0.1.53', 'v1.2.3', '1.2.3-rc.1', 'abcdef0123' * 4):
                    with self.subTest(index=index, accepted=pin):
                        candidate = list(args)
                        candidate[index] = pin
                        result = self.run_cli(*candidate)
                        self.assertEqual(result.returncode, 0, result.stderr)
                        self.assertEqual(json.loads(result.stdout)[
                            'opsVersion' if index == 7 else 'nanosVersion'], pin)
                for pin in ('main', 'master', 'latest', 'nightly', 'stable',
                            'MAIN', 'Latest', ' main', 'main ', '1.2.3 ',
                            '1.2', 'v1', 'abcdef0', 'A' * 40, '01.2.3', ''):
                    with self.subTest(index=index, rejected=pin):
                        candidate = list(args)
                        candidate[index] = pin
                        result = self.run_cli(*candidate)
                        self.assertNotEqual(result.returncode, 0)
                        self.assertEqual(result.stdout, '')

    def test_special_files_and_oversize_rejected_without_hanging(self):
        with tempfile.TemporaryDirectory() as directory:
            artifact = Path(directory) / 'input'
            args = ['--binary', str(artifact), '--source-revision', 'a' * 40,
                    '--amos-revision', 'b' * 40, '--ops-version', '1.2.3',
                    '--nanos-version', '1.2.3']
            os.mkfifo(artifact)
            result = self.run_cli(*args)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('regular file', result.stderr)
            artifact.unlink()
            artifact.symlink_to('/dev/null')
            self.assertIn('regular file', self.run_cli(*args).stderr)
            artifact.unlink()
            with artifact.open('wb') as stream:
                stream.truncate(256 * 1024 * 1024 + 1)
            result = self.run_cli(*args)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('256 MiB', result.stderr)
            self.assertEqual(result.stdout, '')

    def test_concurrent_growth_and_same_size_change_rejected(self):
        for grow in (False, True):
            with self.subTest(grow=grow), tempfile.TemporaryDirectory() as directory:
                artifact = Path(directory) / 'changing-input'
                artifact.write_bytes(b'x' * 64)
                real_read = os.read
                changed = False

                def changing_read(fd, count):
                    nonlocal changed
                    chunk = real_read(fd, count)
                    if not changed:
                        changed = True
                        if grow:
                            with artifact.open('ab') as stream:
                                stream.write(b'growth')
                        else:
                            artifact.write_bytes(b'y' * 64)
                            # Force a detectable metadata change even on coarse clocks.
                            os.utime(artifact, ns=(1, 1))
                    return chunk

                with mock.patch.object(PACKAGE.os, 'read', side_effect=changing_read):
                    with self.assertRaisesRegex(ValueError, 'grew|changed'):
                        PACKAGE.inspect_binary(artifact)

    def test_manifest_is_deterministic_and_unqualified(self):
        with tempfile.TemporaryDirectory() as directory:
            artifact = Path(directory) / 'header-fixture'
            header = bytearray(64)
            header[:6] = b'\x7fELF\x02\x01'
            struct.pack_into('<HH', header, 16, 2, 62)
            header.extend(b'payload beyond structural header')
            artifact.write_bytes(header)
            args = ['--binary', str(artifact), '--source-revision', 'a' * 40,
                    '--amos-revision', 'b' * 40, '--ops-version', '1.2.3',
                    '--nanos-version', '1.2.3']
            first, second = self.run_cli(*args), self.run_cli(*args)
            self.assertEqual(first.returncode, 0, first.stderr)
            self.assertEqual(first.stdout, second.stdout)
            manifest = json.loads(first.stdout)
            self.assertEqual(manifest['status'], 'inputs-inspected-runtime-unqualified')
            self.assertEqual(manifest['binary']['sha256'], hashlib.sha256(header).hexdigest())
            self.assertNotIn(str(artifact), first.stdout)
            header[18] = 183  # AArch64 is rejected for the declared amd64 target.
            artifact.write_bytes(header)
            self.assertNotEqual(self.run_cli(*args).returncode, 0)


if __name__ == '__main__':
    unittest.main()
