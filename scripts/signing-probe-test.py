#!/usr/bin/env python3
"""Hermetic runner signing probe controls; no native key use or Apple request."""
import importlib.util
from contextlib import ExitStack
import os
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import Mock, mock_open, patch

SPEC = importlib.util.spec_from_file_location('preview', Path(__file__).with_name('testflight-preview.py'))
p = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(p)


class SigningProbe(unittest.TestCase):
    """Exercise explicit-key selection, single-call failure, bounds and mode isolation."""

    def setUp(self):
        """Bind tools to disposable fixture storage without executing subprocesses."""
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.tools = Mock(folder=Path(self.tmp.name))
        self.keychain = Path('/SYNTHETIC/signing.keychain-db')

    def test_exact_keychain_sign_and_verify(self):
        """One selected-key sign precedes one non-signing verification."""
        with patch.object(p, 'require_unlocked_keychain') as guard, patch('builtins.open', mock_open(read_data=b'SYNTHETIC OS BINARY')):
            p.signing_probe(self.tools, self.keychain, 'SYNTHETIC_IDENTITY')
        guard.assert_called_once_with(self.keychain)
        calls = self.tools.call.call_args_list
        self.assertEqual(len(calls), 2)
        self.assertEqual(calls[0].args[:1], ('signing-probe',))
        args = calls[0].args[1]
        self.assertEqual(args[:4], ['/usr/bin/codesign', '--force', '--timestamp=none', '--sign'])
        self.assertEqual(args[4:7], ['SYNTHETIC_IDENTITY', '--keychain', str(self.keychain)])
        self.assertEqual(calls[0].kwargs['timeout'], 30)
        self.assertEqual(calls[1].args[1], ['/usr/bin/codesign', '--verify', '--strict', str(self.tools.folder / 'signing-probe')])
        self.assertEqual((self.tools.folder / 'signing-probe').read_bytes(), b'SYNTHETIC OS BINARY')

    def test_key_failure_has_no_second_call(self):
        """Signing failure neither verifies nor automatically retries authentication."""
        self.tools.call.side_effect = p.Blocked('SYNTHETIC key-use failure')
        with patch.object(p, 'require_unlocked_keychain'), patch('builtins.open', mock_open(read_data=b'SYNTHETIC')):
            with self.assertRaises(p.Blocked):
                p.signing_probe(self.tools, self.keychain, 'SYNTHETIC')
        self.assertEqual(self.tools.call.call_count, 1)

    def test_locked_status_stops_before_key_use(self):
        """A refused read-only guard prevents file copying and native signing."""
        with patch.object(p, 'require_unlocked_keychain', side_effect=p.Blocked('SYNTHETIC locked')), patch('builtins.open') as source:
            with self.assertRaises(p.Blocked):
                p.signing_probe(self.tools, self.keychain, 'SYNTHETIC')
        self.tools.call.assert_not_called()
        source.assert_not_called()

    def test_template_bounds_refuse_before_signing(self):
        """Empty and over-budget template bytes cannot reach a signing subprocess."""
        for raw in (b'', b'x' * (4 * 1024 * 1024 + 1)):
            with self.subTest(size=len(raw)), patch.object(p, 'require_unlocked_keychain'), patch('builtins.open', mock_open(read_data=raw)):
                with self.assertRaises(p.Blocked):
                    p.signing_probe(self.tools, self.keychain, 'SYNTHETIC')
        self.tools.call.assert_not_called()

    def test_conflicting_modes_refuse_before_native_admission(self):
        """Direct caller conflicts cannot read credentials or enter keychain admission."""
        with patch.object(p, 'require_unlocked_keychain') as guard:
            with self.assertRaises(p.Blocked):
                p.release(True, True)
        guard.assert_not_called()

    def test_workflow_probe_uses_same_trust_and_secrets_boundary(self):
        """Probe is opt-in in the existing protected manual trusted-main job."""
        workflow = json.loads(Path(__file__).resolve().parent.parent.joinpath('.github/workflows/testflight-preview.yml').read_text())
        self.assertEqual(set(workflow['on']), {'workflow_dispatch'})
        self.assertFalse(workflow['on']['workflow_dispatch']['inputs']['signing_probe_only']['default'])
        job = workflow['jobs']['preview']
        self.assertEqual(job['runs-on'], ['self-hosted', 'macOS', 'prosody'])
        self.assertEqual(job['environment'], 'prosody-asc')
        self.assertIn("github.ref == 'refs/heads/main'", job['if'])
        step = job['steps'][1]
        self.assertEqual(step['env']['PREVIEW_SIGNING_PROBE_ONLY'], '${{ inputs.signing_probe_only }}')
        self.assertIn('--signing-probe-only', step['run'])
        self.assertLess(step['run'].index('exit 1'), step['run'].index('/usr/bin/python3'))



    def release_fixture(self, probe_only, resume=False):
        """Exercise the real release-mode branch with synthetic ASC/native/lease boundaries."""
        root = Path(self.tmp.name)
        (root / 'mobile').mkdir()
        (root / 'mobile/pubspec.yaml').write_text('version: 0.1.0+1\n')
        folder = root / 'private-release'
        folder.mkdir()
        tools = Mock(folder=folder, cleanup_unconfirmed=False)
        tools.call.side_effect = p.Blocked('SYNTHETIC SDK boundary')
        if resume:
            def tool_boundary(phase, argv, **kwargs):
                """Permit verification metadata only, then stop at the export boundary."""
                f = folder / (phase + '.log')
                if phase == 'export-ipa':
                    raise p.Blocked('SYNTHETIC export boundary')
                f.write_text('CDHash=' + 'c' * 40 + '\n' if phase == 'resume-archive-signature-identity' else '26.0\n')
                return f
            tools.call.side_effect = tool_boundary
        asc = Mock()
        asc.credentials.return_value = ('SYNTHETIC_KID', 'SYNTHETIC_ISSUER', b'SYNTHETIC_KEY')
        asc.APP_REQUEST_PATH = '/SYNTHETIC'
        names = ('ASC_API_KEY', 'ASC_ISSUER_ID', 'ASC_KEY_ID', 'ASC_TEAM_ID', 'PROSODY_KEYCHAIN_PASSWORD')
        env = {name: 'SYNTHETIC_SECRET' for name in names}
        env.update(GITHUB_REPOSITORY='narrate-it/prosody', GITHUB_REF='refs/heads/main', RUNNER_OS='macOS', GITHUB_SHA='a' * 40)
        claim = Mock(returncode=0, stdout='WON: R-build-lease ' + 'b' * 40 + '\n')
        release = Mock(returncode=0, stdout='RELEASED: R-build-lease\n')
        with ExitStack() as stack:
            for patcher in [patch.dict(os.environ, env), patch.object(p.os, 'umask'),
                    patch.object(p, 'ROOT', root), patch.object(Path, 'home', return_value=root),
                    patch.object(p, 'require_unlocked_keychain'), patch.object(p, 'load_asc', return_value=asc),
                    patch.object(p, 'api', side_effect=[{'data': [{'id': '123', 'attributes': {'bundleId': p.BUNDLE}}]}, {'data': [], 'included': []}, {'data': []}]),
                    patch.object(p, 'signing_profile', return_value=('SYNTHETIC_IDENTITY', 'SYNTHETIC_PROFILE')),
                    patch.object(p, 'capacity', return_value={'SYNTHETIC': True}),
                    patch.object(p.tempfile, 'mkdtemp', return_value=str(folder)),
                    patch.object(p.subprocess, 'check_output', side_effect=['a' * 40, '', '']),
                    patch.object(p.subprocess, 'run', side_effect=[claim, release]),
                    patch.object(p, 'Tools', return_value=tools)]:
                stack.enter_context(patcher)
            probe = stack.enter_context(patch.object(p, 'signing_probe'))
            persist = stack.enter_context(patch.object(p, 'persist_receipt'))
            if resume:
                original = {'revision': 'a' * 40, 'version': '0.1.0', 'build': '1',
                            'identity': 'SYNTHETIC_IDENTITY', 'profile': 'SYNTHETIC_PROFILE'}
                stack.enter_context(patch.object(p, 'owned_archive', return_value=(root / 'original.xcarchive', original)))
                with self.assertRaisesRegex(p.Blocked, 'SYNTHETIC export boundary'):
                    p.release(False, False, 'prosody-testflight-abcdefgh', 'c' * 40)
            elif probe_only:
                p.release(False, True)
            else:
                with self.assertRaisesRegex(p.Blocked, 'SYNTHETIC SDK boundary'):
                    p.release(False, False)
            self.assertTrue(all(name not in os.environ for name in names))
        return tools, probe, persist

    def test_probe_only_stops_before_sdk_build_and_upload(self):
        """The real probe-only branch returns before any SDK/build/archive/upload tool call."""
        tools, probe, persist = self.release_fixture(True)
        probe.assert_called_once()
        tools.call.assert_not_called()
        receipt = persist.call_args.args[1]
        self.assertEqual(receipt['signingProbe'], 'PASSED')
        self.assertEqual(receipt['uploadState'], 'NOT ATTEMPTED')

    def test_normal_release_continues_after_probe(self):
        """Normal mode retains the SDK/build path after the same prerequisite probe."""
        tools, probe, persist = self.release_fixture(False)
        probe.assert_called_once()
        self.assertEqual(tools.call.call_args.args[0], 'sdk-version')
        self.assertEqual(tools.call.call_count, 1)
        self.assertEqual(persist.call_args.args[1]['uploadState'], 'NOT ATTEMPTED')



    def test_resume_skips_flutter_and_archive_build(self):
        """The real resume branch reaches export after revalidation without any rebuild call."""
        tools, probe, persist = self.release_fixture(False, resume=True)
        probe.assert_called_once()
        phases = [call.args[0] for call in tools.call.call_args_list]
        self.assertEqual(phases, ['sdk-version', 'archive-metadata', 'archive-signature',
                                  'resume-archive-signature-identity', 'export-ipa'])
        self.assertEqual(persist.call_args.args[1]['uploadState'], 'NOT ATTEMPTED')
        self.assertEqual(persist.call_args.args[1]['resumedArchive']['cdhash'], 'c' * 40)


if __name__ == '__main__':
    unittest.main(verbosity=2)
