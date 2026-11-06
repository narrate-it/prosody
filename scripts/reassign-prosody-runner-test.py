#!/usr/bin/env python3
"""Hermetic registration state-machine checks; no GitHub or launchd effects."""
import importlib.util
import json
import os
from pathlib import Path
import plistlib
import signal
import subprocess
import tempfile
import unittest
from unittest import mock

SPEC = importlib.util.spec_from_file_location('runner_reassign', Path(__file__).with_name('reassign-prosody-runner.py'))
RUNNER = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(RUNNER)


class Reassignment(unittest.TestCase):
    """Exercise source guards and partial-failure behavior with synthetic adapters."""
    def setUp(self):
        """Create an isolated owned installation and synthetic stopped service."""
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.home = Path(self.temp.name)
        self.root = self.home / 'actions-runner-prosody'
        self.root.mkdir()
        (self.root / 'bin').mkdir()
        for name in ['config.sh', 'svc.sh', 'bin/Runner.Listener', 'runsvc.sh']:
            (self.root / name).write_text('synthetic fixture')
        self.local = {'agentName': RUNNER.NAME, 'agentId': 2, 'gitHubUrl': 'https://github.com/sirerun/prosody'}
        self.write_binding()
        self.service_file = self.home / ('Library/LaunchAgents/actions.runner.sirerun-prosody.' + RUNNER.NAME + '.plist')
        self.write_service(self.service_file)
        self.custody = self.home / '.local/state/prosody-runner-reassignment'
        self.effects = []
        self.posts = []
        self.target = False
        self.is_loaded = False
        self.busy = False
        self.collision = False
        self.deny_registration = False
        self.stale_server = False
        self.deny_source_inventory = False
        self.process_names = ''
        self.sync_count = 0
        self.require_sync_count = None
        self.required_parent_syncs = set()
        self.synced_directories = set()
        for patcher in [mock.patch.object(Path, 'home', return_value=self.home), mock.patch.object(RUNNER.sys, 'platform', 'darwin'), mock.patch.object(RUNNER.socket, 'gethostname', return_value='fixture-host'), mock.patch.dict(os.environ, {'PROSODY_RUNNER_EXPECTED_HOST': 'fixture-host'}), mock.patch.object(RUNNER, 'github', side_effect=self.github), mock.patch.object(RUNNER, 'command', side_effect=self.command)]:
            patcher.start()
            self.addCleanup(patcher.stop)

    def write_binding(self):
        """Persist only synthetic noncredential registration facts."""
        (self.root / '.runner').write_text(json.dumps(self.local))

    def write_service(self, path):
        """Create the same narrow service shape inspected by the reconciler."""
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(plistlib.dumps({'Label': path.stem, 'WorkingDirectory': str(self.root), 'ProgramArguments': [str(self.root / 'runsvc.sh')]}))
        (self.root / '.service').write_text(str(path))

    def github(self, path, method='GET'):
        """Model token permission and registration transitions without network access."""
        if method == 'POST':
            self.posts.append(path)
            if path.endswith('registration-token') and self.deny_registration:
                raise RUNNER.Refused('synthetic permission denial')
            return {'token': 'SYNTHETIC_REMOVE' if path.endswith('remove-token') else 'SYNTHETIC_REGISTER'}
        if path.startswith('repos/' + RUNNER.OLD + '/'):
            if self.deny_source_inventory:
                raise RUNNER.Refused('synthetic source inventory denial')
            rows = [] if self.stale_server else [{'id': 2, 'name': RUNNER.NAME, 'busy': self.busy, 'status': 'offline'}]
            return {'total_count': len(rows), 'runners': rows}
        row = {'id': 10, 'name': RUNNER.NAME, 'busy': False, 'status': 'online' if self.is_loaded else 'offline', 'labels': [{'name': s} for s in ['self-hosted', 'macOS', 'X64', 'prosody']]}
        if path.endswith('?per_page=100'):
            rows = [row] if self.target or self.collision else []
            return {'total_count': len(rows), 'runners': rows}
        return row

    def command(self, args, cwd=None, env=None):
        """Check secret transport and model official config/service command effects."""
        if args == ['launchctl', 'list']:
            return '-\t0\t' + self.service_file.stem if self.is_loaded else ''
        if args == ['ps', '-axo', 'comm=']:
            return self.process_names
        self.effects.append(args)
        if args[0] == str(self.root / 'config.sh'):
            token = env['ACTIONS_RUNNER_INPUT_TOKEN'] if env else None
            if token:
                self.assertNotIn(token, args)
            if 'remove' in args:
                if self.require_sync_count is not None:
                    self.assertEqual(self.sync_count, self.require_sync_count)
                self.assertTrue(self.required_parent_syncs.issubset(self.synced_directories))
                self.assertEqual(token, None if self.stale_server else 'SYNTHETIC_REMOVE')
                self.assertEqual('--local' in args, self.stale_server)
                (self.root / '.runner').unlink()
            else:
                self.assertEqual(token, 'SYNTHETIC_REGISTER')
                self.target = True
                self.assertEqual(args[args.index('--work') + 1], RUNNER.WORK)
                self.local = {'agentName': RUNNER.NAME, 'agentId': 10, 'gitHubUrl': RUNNER.TARGET_URL, 'workFolder': RUNNER.WORK}
                self.write_binding()
        elif args[-1] == 'install':
            self.service_file = self.home / ('Library/LaunchAgents/actions.runner.narrate-it-prosody.' + RUNNER.NAME + '.plist')
            self.write_service(self.service_file)
        elif args[-1] == 'start':
            self.is_loaded = True
        else:
            self.fail('unexpected synthetic command')
        return ''

    def test_plan_is_read_only(self):
        """A valid plan performs no token issuance or local service changes."""
        result = RUNNER.reconcile(self.root, self.custody)
        self.assertEqual(result['effect'], 'none')
        self.assertEqual(self.posts, [])
        self.assertEqual(self.effects, [])
        self.assertFalse(self.custody.exists())

    def test_reassign_and_idempotent_verification(self):
        """Reassignment verifies new identity; repeats never rotate credentials again."""
        legacy = self.root / '_work'
        legacy.mkdir()
        (legacy / 'keep').write_text('preserved legacy workspace')
        result = RUNNER.reconcile(self.root, self.custody, True)
        self.assertEqual(result['status'], 'online')
        self.assertEqual(result['id'], 10)
        self.assertEqual(len(self.posts), 2)
        before = list(self.effects)
        RUNNER.reconcile(self.root, self.custody, True)
        self.assertEqual(self.effects, before)
        self.assertEqual(len(self.posts), 2)
        record = (self.custody / 'reassignment.json').read_text()
        self.assertEqual(json.loads(record)['originalID'], 2)
        self.assertNotIn('SYNTHETIC_REMOVE', record)
        self.assertNotIn('SYNTHETIC_REGISTER', record)
        self.assertEqual((legacy / 'keep').read_text(), 'preserved legacy workspace')

    def test_permission_denial_preserves_old_configuration(self):
        """Both token permissions must succeed before removing the stopped service."""
        self.deny_registration = True
        with self.assertRaises(RUNNER.Refused):
            RUNNER.reconcile(self.root, self.custody, True)
        self.assertTrue(self.service_file.exists())
        self.assertTrue((self.root / '.service').exists())
        self.assertEqual(RUNNER.binding(self.root)['agentId'], 2)
        self.assertEqual(self.effects, [])

    def test_busy_and_collision_refusals(self):
        """Preserve an active old runner or an unrelated target name."""
        for kind in ['busy', 'collision']:
            with self.subTest(kind=kind):
                self.busy, self.collision = kind == 'busy', kind == 'collision'
                with self.assertRaises(RUNNER.Refused):
                    RUNNER.reconcile(self.root, self.custody, True)
                self.assertEqual(self.posts, [])
                self.assertEqual(self.effects, [])

    def test_foreign_service_refused(self):
        """A LaunchAgent pointing elsewhere is never removed or started."""
        data = plistlib.loads(self.service_file.read_bytes())
        data['WorkingDirectory'] = str(self.home / 'other')
        self.service_file.write_bytes(plistlib.dumps(data))
        with self.assertRaises(RUNNER.Refused):
            RUNNER.reconcile(self.root, self.custody, True)
        self.assertEqual(self.posts, [])
        self.assertTrue(self.service_file.exists())

    def test_loaded_legacy_service_refused(self):
        """Never remove a loaded legacy service even if the API says offline."""
        self.is_loaded = True
        with self.assertRaises(RUNNER.Refused):
            RUNNER.reconcile(self.root, self.custody, True)
        self.assertEqual(self.posts, [])
        self.assertTrue(self.service_file.exists())

    def test_foreign_host_refused(self):
        """An existing directory on another Mac is not admitted by path alone."""
        with mock.patch.object(RUNNER.socket, 'gethostname', return_value='different-host'):
            with self.assertRaises(RUNNER.Refused):
                RUNNER.reconcile(self.root, self.custody, True)
        self.assertEqual(self.posts, [])

    def test_local_process_refused(self):
        """API-offline state does not authorize removing a locally running worker."""
        for name in ['Runner.Listener', '/synthetic/bin/Runner.Worker']:
            with self.subTest(name=name):
                self.process_names = name
                with self.assertRaises(RUNNER.Refused):
                    RUNNER.reconcile(self.root, self.custody, True)
                self.assertEqual(self.posts, [])
                self.assertEqual(self.effects, [])

    def test_checkpoint_sync_precedes_removal(self):
        """Both atomic checkpoint and retained service backup sync before removal."""
        real_sync = RUNNER.sync_directory

        def synced(path):
            """Run the real directory fsync while recording its ordering."""
            real_sync(path)
            self.synced_directories.add(path)
            if path == self.custody and (self.custody / 'reassignment.json').exists():
                self.sync_count += 1

        self.require_sync_count = 2
        self.required_parent_syncs = {self.home, self.home / '.local', self.home / '.local/state'}
        with mock.patch.object(RUNNER, 'sync_directory', side_effect=synced):
            RUNNER.reconcile(self.root, self.custody, True)

    def test_existing_target_work_directory_refused(self):
        """Reassignment never adopts a preexisting work directory silently."""
        (self.root / RUNNER.WORK).mkdir()
        with self.assertRaises(RUNNER.Refused):
            RUNNER.reconcile(self.root, self.custody, True)
        self.assertEqual(self.posts, [])

    def test_verified_stale_source_uses_only_local_cleanup(self):
        """Successful empty source inventory permits no-server local config removal."""
        self.stale_server = True
        result = RUNNER.reconcile(self.root, self.custody, True)
        self.assertEqual(result['status'], 'online')
        self.assertEqual(self.posts, ['repos/' + RUNNER.TARGET + '/actions/runners/registration-token'])
        self.assertTrue(json.loads((self.custody / 'reassignment.json').read_text())['sourceServerAbsent'])

    def test_source_inventory_failure_never_means_absence(self):
        """Denied source reads never authorize local cleanup or token issuance."""
        self.deny_source_inventory = True
        with self.assertRaises(RUNNER.Refused):
            RUNNER.reconcile(self.root, self.custody, True)
        self.assertEqual(self.posts, [])
        self.assertEqual(self.effects, [])
        self.assertTrue((self.root / '.runner').exists())


class SubprocessCleanup(unittest.TestCase):
    """Check cancellation paths without creating or killing a real process."""
    def test_interruption_and_timeout_reap_owned_group(self):
        """Both ordinary timeout and interruption terminate the exact owned group."""
        for problem, leader_exited in [(KeyboardInterrupt(), False), (subprocess.TimeoutExpired('synthetic', 1), False), (subprocess.TimeoutExpired('synthetic', 1), True)]:
            with self.subTest(problem=type(problem).__name__, leader_exited=leader_exited):
                process = mock.Mock(pid=12345)
                process.poll.return_value = 0 if leader_exited else None
                process.communicate.side_effect = [problem, ('', '')]
                process.returncode = 0
                with mock.patch.object(RUNNER.subprocess, 'Popen', return_value=process) as spawn, mock.patch.object(RUNNER.os, 'killpg') as kill, mock.patch.object(RUNNER, 'DEADLINE', RUNNER.time.monotonic() + 10):
                    with self.assertRaises((KeyboardInterrupt, RUNNER.Refused)):
                        RUNNER.command(['synthetic'])
                    kill.assert_called_once_with(12345, signal.SIGTERM)
                    self.assertEqual(process.communicate.call_count, 2)
                    self.assertTrue(spawn.call_args.kwargs['start_new_session'])


if __name__ == '__main__':
    unittest.main()
