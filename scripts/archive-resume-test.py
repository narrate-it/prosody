#!/usr/bin/env python3
"""Hermetic archive-resume custody and binding checks; no native signing or Apple calls."""
import importlib.util
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import Mock, patch

SPEC = importlib.util.spec_from_file_location('preview', Path(__file__).with_name('testflight-preview.py'))
p = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(p)


class ArchiveResume(unittest.TestCase):
    """Keep replay and artifact selection scoped to unuploaded current app inputs."""

    def setUp(self):
        """Create a private synthetic cache and original completed-job receipt."""
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.home = Path(self.tmp.name)
        self.name = 'prosody-testflight-abcdefgh'
        self.cache = self.home / 'Library/Caches' / self.name
        self.app = self.cache / 'AgentProsody.xcarchive/Products/Applications/Runner.app'
        self.app.mkdir(parents=True)
        self.cache.chmod(0o700)
        self.receipt = {'revision': 'a' * 40, 'uploadState': 'NOT ATTEMPTED', 'signingProbe': 'PASSED',
                        'version': '1.0.1', 'build': '41', 'identity': 'IDENTITY', 'profile': 'PROFILE'}
        self.write_receipt()
        for obj in (patch.object(Path, 'home', return_value=self.home),
                    patch.object(p.subprocess, 'run', return_value=Mock(returncode=0))):
            obj.start()
            self.addCleanup(obj.stop)

    def write_receipt(self):
        """Persist fixture metadata as a single-link private regular file."""
        f = self.cache / 'receipt.json'
        f.write_text(json.dumps(self.receipt))
        f.chmod(0o600)

    def load(self):
        """Admit the fixed fixture reference against a synthetic current revision."""
        return p.owned_archive(self.name, 'c' * 40, 'b' * 40)

    def test_owned_unuploaded_archive(self):
        """An owned receipt with matching ancestor/mobile checks returns its fixed archive."""
        archive, prior = self.load()
        self.assertEqual(archive, self.app.parents[2])
        self.assertEqual(prior, self.receipt)
        calls = p.subprocess.run.call_args_list
        self.assertEqual(calls[0].args[0], ['git', 'merge-base', '--is-ancestor', 'a' * 40, 'b' * 40])
        self.assertEqual(calls[1].args[0], ['git', 'diff', '--quiet', 'a' * 40, 'b' * 40, '--', 'mobile'])

    def test_reference_denials(self):
        """Traversal, absolute paths and invalid hashes cannot choose another cache."""
        for name, digest in [('../outside', 'c' * 40), ('/tmp/other', 'c' * 40), (self.name, 'BAD')]:
            with self.subTest(name=name), self.assertRaises(p.Blocked):
                p.owned_archive(name, digest, 'b' * 40)

    def test_unknown_or_accepted_never_reuploads(self):
        """Remote uncertainty or prior acceptance requires reconciliation instead of resume."""
        for state in ('UNKNOWN', 'ACCEPTED', None):
            self.receipt['uploadState'] = state
            self.write_receipt()
            with self.subTest(state=state), self.assertRaises(p.Blocked):
                self.load()

    def test_source_drift_refuses(self):
        """Nonancestor source or changed mobile input fails before artifact export."""
        for results in ([Mock(returncode=1)], [Mock(returncode=0), Mock(returncode=1)]):
            with patch.object(p.subprocess, 'run', side_effect=results):
                with self.assertRaises(p.Blocked):
                    self.load()

    def test_linked_cache_refuses(self):
        """The cache root cannot redirect to a foreign directory."""
        moved = self.cache.with_name('saved')
        self.cache.rename(moved)
        self.cache.symlink_to(moved)
        with self.assertRaises(OSError):
            self.load()

    def test_linked_receipt_refuses(self):
        """A linked receipt cannot redirect even to same-user data."""
        f = self.cache / 'receipt.json'
        target = self.home / 'other'
        f.rename(target)
        f.symlink_to(target)
        with self.assertRaises(OSError):
            self.load()

    def test_fifo_receipt_refuses_without_block(self):
        """A FIFO receipt is opened nonblocking and rejected as nonregular."""
        f = self.cache / 'receipt.json'
        f.unlink()
        os.mkfifo(f, 0o600)
        with self.assertRaises(p.Blocked):
            self.load()

    def test_public_receipt_and_cache_refuse(self):
        """World-readable custody is not accepted as private release input."""
        (self.cache / 'receipt.json').chmod(0o644)
        with self.assertRaises(p.Blocked):
            self.load()
        (self.cache / 'receipt.json').chmod(0o600)
        self.cache.chmod(0o755)
        with self.assertRaises(p.Blocked):
            self.load()

    def test_binding_refuses_stale_selection(self):
        """Version/build collisions or changed profile/identity cannot be silently reused."""
        p.resume_binding(self.receipt, '1.0.1', '41', 'IDENTITY', 'PROFILE')
        for key in ('version', 'build', 'identity', 'profile'):
            with self.subTest(key=key), self.assertRaises(p.Blocked):
                p.resume_binding({**self.receipt, key: 'CHANGED'}, '1.0.1', '41', 'IDENTITY', 'PROFILE')

    def test_cdhash_requires_one_exact_match(self):
        """Missing, changed or ambiguous signed identities cannot reach export."""
        f = self.home / 'signature.log'
        tools = Mock()
        tools.call.return_value = f
        for value in ('', 'CDHash=' + 'd' * 40 + '\n', ('CDHash=' + 'c' * 40 + '\n') * 2):
            f.write_text(value)
            with self.subTest(value=value), self.assertRaises(p.Blocked):
                p.check_archive_cdhash(tools, self.app.parents[2], 'c' * 40)
        f.write_text('CDHash=' + 'c' * 40 + '\n')
        p.check_archive_cdhash(tools, self.app.parents[2], 'c' * 40)

    def test_conflicting_modes_never_unlock(self):
        """Missing paired references and probe conflicts stop before native admission."""
        with patch.object(p, 'require_unlocked_keychain') as native:
            for args in [(False, True, self.name, 'c' * 40), (True, False, self.name, '')]:
                with self.subTest(args=args), self.assertRaises(p.Blocked):
                    p.release(*args)
            native.assert_not_called()



    def attempt_fixture(self):
        """Prepare one synthetic continuation with no external upload transport."""
        folder = self.home / 'new-private-job'
        folder.mkdir()
        receipt = {'revision': 'b' * 40, 'uploadState': 'NOT ATTEMPTED',
                   'ipaSHA256': 'e' * 64, 'resumedArchive': {'cdhash': 'c' * 40}}
        return folder, receipt, Mock(), self.app.parents[2]

    def test_lost_response_blocks_second_resume_despite_stale_apple(self):
        """A lost response durably blocks origin admission before any stale Apple lookup."""
        folder, receipt, tools, archive = self.attempt_fixture()
        tools.call.side_effect = p.Blocked('SYNTHETIC lost upload response')
        with self.assertRaises(p.Blocked):
            p.upload_resumed_once(tools, folder, receipt, folder / 'app.ipa', [], archive, self.receipt)
        self.assertEqual(tools.call.call_count, 1)
        original = json.loads((self.cache / 'receipt.json').read_text())
        self.assertEqual(original['uploadState'], 'UNKNOWN')
        self.assertEqual(original['resumeAttempt']['cache'], folder.name)
        with patch.object(p, 'api', return_value={'data': []}) as api:
            with self.assertRaises(p.Blocked):
                self.load()
            api.assert_not_called()
        with self.assertRaises(p.Blocked):
            p.upload_resumed_once(tools, folder, receipt, folder / 'app.ipa', [], archive, self.receipt)
        self.assertEqual(tools.call.call_count, 1)

    def test_accepted_origin_blocks_another_attempt(self):
        """Accepted transport updates both receipts while retaining the same stable pointer."""
        folder, receipt, tools, archive = self.attempt_fixture()
        p.upload_resumed_once(tools, folder, receipt, folder / 'app.ipa', [], archive, self.receipt)
        self.assertTrue(receipt['uploadState'].startswith('ACCEPTED:'))
        self.assertEqual(json.loads((self.cache / 'receipt.json').read_text())['uploadState'], receipt['uploadState'])
        with self.assertRaises(p.Blocked):
            self.load()
        self.assertEqual(tools.call.call_count, 1)

    def test_overlapping_upload_refuses_before_state_or_transport(self):
        """An active real artifact lock prevents a second attempt from overwriting admission."""
        folder, receipt, tools, archive = self.attempt_fixture()
        with p.archive_upload_lock(self.cache):
            with self.assertRaises(p.Blocked):
                p.upload_resumed_once(tools, folder, receipt, folder / 'app.ipa', [], archive, self.receipt)
        tools.call.assert_not_called()
        self.assertEqual(json.loads((self.cache / 'receipt.json').read_text())['uploadState'], 'NOT ATTEMPTED')

    def test_duplicate_receipt_state_refuses(self):
        """Conflicting duplicated upload states cannot erase uncertainty."""
        (self.cache / 'receipt.json').write_text('{"uploadState":"UNKNOWN","uploadState":"NOT ATTEMPTED"}')
        with self.assertRaises(p.Blocked):
            self.load()


if __name__ == '__main__':
    unittest.main(verbosity=2)
