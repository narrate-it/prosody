#!/usr/bin/env python3
"""Hermetic session-only reconciliation tests; no launchd, keychain or GitHub effects."""
import importlib.util
import json
import os
from pathlib import Path
import plistlib
import tempfile
import unittest
from unittest import mock

SPEC = importlib.util.spec_from_file_location('gui_session', Path(__file__).with_name('runner-gui-session.py'))
GUI = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(GUI)


class SessionControl(unittest.TestCase):
    """Exercise custody, drift, idle admission and partial-failure behavior."""

    def setUp(self):
        """Create an isolated synthetic registration and LaunchAgent."""
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.home = Path(self.tmp.name)
        self.root = self.home / 'actions-runner-prosody'
        self.root.mkdir()
        self.custody = self.home / '.local/state/prosody-runner-gui-session'
        self.path = self.home / 'Library/LaunchAgents/actions.runner.narrate-it-prosody.prosody-laptop-runner.plist'
        self.path.parent.mkdir(parents=True)
        self.label = self.path.stem
        self.original = plistlib.dumps({'Label': self.label, 'SessionCreate': True,
            'WorkingDirectory': str(self.root), 'ProgramArguments': [str(self.root / 'runsvc.sh')],
            'EnvironmentVariables': {'ACTIONS_RUNNER_SVC': '1'}})
        self.path.write_bytes(self.original)
        (self.root / '.service').write_text(str(self.path))
        self.local = {'agentId': 2, 'agentName': GUI.OWNER.NAME, 'gitHubUrl': GUI.OWNER.TARGET_URL,
                      'workFolder': GUI.OWNER.WORK}
        (self.root / '.runner').write_text(json.dumps(self.local))
        self.row = {'id': 2, 'name': GUI.OWNER.NAME, 'busy': False, 'status': 'online',
                    'labels': [{'name': n} for n in ('self-hosted', 'macOS', 'X64', 'prosody')]}
        self.effects = []
        self.workers = ''
        self.fail_bootstrap = False
        self.fail_stop = False
        self.is_loaded = True
        self.gui_lookup_original = GUI.gui_service_absent
        self.command_patch_original = GUI.OWNER.command
        for patch in [mock.patch.object(Path, 'home', return_value=self.home),
                      mock.patch.object(GUI.sys, 'platform', 'darwin'),
                      mock.patch.object(GUI.os, 'getuid', return_value=self.path.stat().st_uid),
                      mock.patch.object(GUI.socket, 'gethostname', return_value='SYNTHETIC'),
                      mock.patch.dict(os.environ, {'PROSODY_RUNNER_EXPECTED_HOST': 'SYNTHETIC'}),
                      mock.patch.object(GUI.OWNER, 'repository_runners', side_effect=lambda _: [self.row.copy()]),
                      mock.patch.object(GUI.OWNER, 'command', side_effect=self.command),
                      mock.patch.object(GUI.time, 'sleep'),
                      mock.patch.object(GUI, 'gui_service_absent', side_effect=lambda *args: not self.is_loaded)]:
            patch.start()
            self.addCleanup(patch.stop)

    def command(self, args, **unused):
        """Model only launchd inspection/restart and process discovery."""
        if args == ['ps', '-axo', 'comm=']:
            return self.workers
        if args == ['launchctl', 'list']:
            return '- 0 ' + self.label if self.is_loaded else ''
        if args[:2] == ['launchctl', 'print']:
            return 'SYNTHETIC GUI DOMAIN'
        self.effects.append(args)
        if args[1] == 'bootout':
            self.is_loaded = False
        elif args[1] == 'bootstrap' and not self.fail_bootstrap:
            self.is_loaded = True
            self.row['status'] = 'online'
        if self.fail_stop and args[1] == 'bootout':
            raise GUI.OWNER.Refused('SYNTHETIC stop failure')
        if self.fail_bootstrap and args[1] == 'bootstrap':
            raise GUI.OWNER.Refused('SYNTHETIC start failure')
        return ''

    def test_plan_has_no_effects(self):
        """Planning preserves source bytes and creates no custody."""
        result = GUI.reconcile(self.root, self.custody)
        self.assertEqual(result['effect'], 'none')
        self.assertFalse(self.effects)
        self.assertEqual(self.path.read_bytes(), self.original)
        self.assertFalse(self.custody.exists())

    def test_apply_changes_only_session_boolean(self):
        """Apply retains exact backup and every unrelated plist property."""
        result = GUI.reconcile(self.root, self.custody, True)
        want = plistlib.loads(self.original)
        want['SessionCreate'] = False
        self.assertEqual(plistlib.loads(self.path.read_bytes()), want)
        self.assertEqual(result['signing'], 'NOT TESTED')
        self.assertEqual(len(self.effects), 2)
        self.assertEqual(list(self.custody.glob('launchagent-*.plist'))[0].read_bytes(), self.original)

    def test_idempotent_has_no_restart(self):
        """An already configured service is verified without restart."""
        self.path.write_bytes(GUI.desired_bytes(self.original))
        GUI.reconcile(self.root, self.custody, True)
        self.assertFalse(self.effects)

    def test_busy_refuses(self):
        """A current remote busy decision prevents stop."""
        self.row['busy'] = True
        with self.assertRaises(GUI.OWNER.Refused):
            GUI.reconcile(self.root, self.custody, True)
        self.assertFalse(self.effects)

    def test_local_worker_refuses(self):
        """A worker observed despite remote idle prevents stop."""
        self.workers = str(self.root / 'bin/Runner.Worker')
        with self.assertRaises(GUI.OWNER.Refused):
            GUI.reconcile(self.root, self.custody, True)
        self.assertFalse(self.effects)

    def test_new_remote_busy_refuses(self):
        """Busy state arriving between plan and stop is rechecked."""
        with mock.patch.object(GUI.OWNER, 'repository_runners', side_effect=[
                [self.row], [{**self.row, 'busy': True}]]):
            with self.assertRaises(GUI.OWNER.Refused):
                GUI.reconcile(self.root, self.custody, True)
        self.assertFalse(self.effects)
        self.assertFalse((self.custody / 'session-change.json').exists())
        result = GUI.reconcile(self.root, self.custody, True)
        self.assertEqual(result['effect'], 'runner-restarted')

    def test_stop_failure_preserves_definition(self):
        """An ambiguous stop leaves original bytes and a recoverable phase."""
        self.fail_stop = True
        with self.assertRaises(GUI.OWNER.Refused):
            GUI.reconcile(self.root, self.custody, True)
        self.assertEqual(self.path.read_bytes(), self.original)
        self.assertEqual(json.loads((self.custody / 'session-change.json').read_text())['phase'], 'stop-started')

    def test_bootstrap_failure_is_not_rollback(self):
        """Failed start retains changed definition and checkpoint without implicit retry."""
        self.fail_bootstrap = True
        with self.assertRaises(GUI.OWNER.Refused):
            GUI.reconcile(self.root, self.custody, True)
        self.assertFalse(plistlib.loads(self.path.read_bytes())['SessionCreate'])
        self.assertEqual(json.loads((self.custody / 'session-change.json').read_text())['phase'], 'definition-written')
        self.assertEqual(len(self.effects), 2)

    def test_malformed_session_denials(self):
        """Missing and non-Boolean fields cannot silently widen session access."""
        for value in ({}, {'SessionCreate': 'true'}, {'SessionCreate': 1}, []):
            with self.subTest(value=value), self.assertRaises(GUI.OWNER.Refused):
                GUI.desired_bytes(plistlib.dumps(value))

    def test_foreign_registration_refuses(self):
        """An unrelated repository cannot be modified."""
        self.local['gitHubUrl'] = 'https://github.com/example/foreign'
        (self.root / '.runner').write_text(json.dumps(self.local))
        with self.assertRaises(GUI.OWNER.Refused):
            GUI.reconcile(self.root, self.custody, True)
        self.assertFalse(self.effects)

    def test_symlink_service_refuses(self):
        """A linked service definition is never replaced."""
        saved = self.home / 'saved.plist'
        self.path.replace(saved)
        self.path.symlink_to(saved)
        with self.assertRaises(GUI.OWNER.Refused):
            GUI.reconcile(self.root, self.custody, True)
        self.assertFalse(self.effects)

    def test_foreign_backup_refuses(self):
        """A mismatched retained backup is not overwritten."""
        GUI.OWNER.prepare_custody(self.custody)
        digest = GUI.hashlib.sha256(self.original).hexdigest()
        (self.custody / ('launchagent-' + digest + '.plist')).write_text('FOREIGN')
        with self.assertRaises(GUI.OWNER.Refused):
            GUI.reconcile(self.root, self.custody, True)
        self.assertFalse(self.effects)


    def test_unfinished_checkpoint_refuses(self):
        """An uncertain prior stop cannot be overwritten by another attempt."""
        GUI.OWNER.prepare_custody(self.custody)
        record = self.custody / 'session-change.json'
        raw = json.dumps({'name': GUI.OWNER.NAME, 'runnerID': 2, 'phase': 'stop-started'})
        record.write_text(raw)
        with self.assertRaises(GUI.OWNER.Refused):
            GUI.reconcile(self.root, self.custody, True)
        self.assertEqual(record.read_text(), raw)
        self.assertFalse(self.effects)

    def test_successful_checkpoint_idempotency(self):
        """A verified complete transition can be inspected without another restart."""
        GUI.reconcile(self.root, self.custody, True)
        before = list(self.effects)
        GUI.reconcile(self.root, self.custody, True)
        self.assertEqual(self.effects, before)

    def test_sigterm_unwinds_owned_command(self):
        """The installed termination handler reaches the existing child cleanup finally."""
        child = mock.Mock()
        child.pid = 12345
        child.communicate.side_effect = [GUI.OWNER.Refused('SYNTHETIC interruption'), ('', '')]
        with mock.patch.object(GUI.OWNER.subprocess, 'Popen', return_value=child), \
                mock.patch.object(GUI.OWNER.os, 'killpg') as kill:
            # Bypass the synthetic command adapter to test the real owned-process control.
            real_command = self.command_patch_original
            with self.assertRaises(GUI.OWNER.Refused):
                real_command(['SYNTHETIC'])
            kill.assert_called_once_with(12345, GUI.signal.SIGTERM)
        with self.assertRaises(GUI.OWNER.Refused):
            GUI.interrupted(GUI.signal.SIGTERM, None)



    def test_exclusive_operation_refuses_overlap(self):
        """A held real lock refuses a second apply before any remote or service observation."""
        with GUI.operation_lock(self.custody):
            with mock.patch.object(GUI.OWNER, 'repository_runners') as inventory:
                with self.assertRaises(GUI.OWNER.Refused):
                    GUI.reconcile(self.root, self.custody, True)
                inventory.assert_not_called()
        self.assertFalse(self.effects)
        self.assertFalse((self.custody / 'session-change.json').exists())
        self.assertEqual(GUI.reconcile(self.root, self.custody, True)['effect'], 'runner-restarted')

    def test_operation_lock_survives_release(self):
        """Closing an operation leaves the same stable inode for later mutual exclusion."""
        with GUI.operation_lock(self.custody):
            inode = (self.custody / 'operation.lock').stat().st_ino
        with GUI.operation_lock(self.custody):
            self.assertEqual((self.custody / 'operation.lock').stat().st_ino, inode)

    def test_linked_lock_refuses(self):
        """A pre-existing lock symlink cannot redirect control-file writes."""
        GUI.OWNER.prepare_custody(self.custody)
        (self.custody / 'operation.lock').symlink_to(self.home / 'foreign')
        with self.assertRaises(OSError):
            GUI.reconcile(self.root, self.custody, True)
        self.assertFalse(self.effects)
        self.assertFalse((self.home / 'foreign').exists())



    def stopped_fixture(self):
        """Reproduce the inspected post-stop state with retained exact service bytes."""
        with mock.patch.object(GUI, 'wait_stopped', side_effect=GUI.OWNER.Refused('SYNTHETIC late listener')):
            with self.assertRaises(GUI.OWNER.Refused):
                GUI.reconcile(self.root, self.custody, True)
        self.assertEqual(json.loads((self.custody / 'session-change.json').read_text())['phase'], 'stopped')
        self.row['status'] = 'offline'
        self.effects.clear()

    def test_graceful_stop_waits_without_second_bootout(self):
        """A listener seen just after bootout may exit normally without aborting the change."""
        original = self.command
        calls = 0

        def processes(args, **kwargs):
            """Report one transient listener only after the first stop."""
            nonlocal calls
            if args == ['ps', '-axo', 'comm='] and not self.is_loaded:
                calls += 1
                return str(self.root / 'bin/Runner.Listener') if calls == 1 else ''
            return original(args, **kwargs)

        with mock.patch.object(GUI.OWNER, 'command', side_effect=processes):
            GUI.reconcile(self.root, self.custody, True)
        self.assertEqual(calls, 2)
        self.assertEqual(sum(a[1] == 'bootout' for a in self.effects), 1)

    def test_resume_exact_stopped_transition(self):
        """Explicit recovery completes the original change without issuing another stop."""
        self.stopped_fixture()
        result = GUI.reconcile(self.root, self.custody, resume=True)
        self.assertEqual(result['signing'], 'NOT TESTED')
        self.assertFalse(plistlib.loads(self.path.read_bytes())['SessionCreate'])
        self.assertEqual([a[1] for a in self.effects], ['bootstrap'])
        self.assertEqual(json.loads((self.custody / 'session-change.json').read_text())['phase'], 'online; signing-unqualified')

    def test_resume_refuses_changed_original(self):
        """A changed stopped definition cannot reuse the retained recovery receipt."""
        self.stopped_fixture()
        value = plistlib.loads(self.path.read_bytes())
        value['ForeignChange'] = True
        self.path.write_bytes(plistlib.dumps(value))
        with self.assertRaises(GUI.OWNER.Refused):
            GUI.reconcile(self.root, self.custody, resume=True)
        self.assertFalse(self.effects)

    def test_resume_refuses_loaded_service(self):
        """Recovery cannot restart a service already adopted or started elsewhere."""
        self.stopped_fixture()
        self.is_loaded = True
        with self.assertRaises(GUI.OWNER.Refused):
            GUI.reconcile(self.root, self.custody, resume=True)
        self.assertFalse(self.effects)

    def test_resume_refuses_other_phase(self):
        """Only the inspected stopped phase is eligible; uncertain writes stay closed."""
        self.stopped_fixture()
        p = self.custody / 'session-change.json'
        d = json.loads(p.read_text())
        d['phase'] = 'definition-written'
        p.write_text(json.dumps(d))
        with self.assertRaises(GUI.OWNER.Refused):
            GUI.reconcile(self.root, self.custody, resume=True)
        self.assertFalse(self.effects)

    def test_resume_refuses_bad_backup(self):
        """A missing or mismatched original backup prevents recovery effects."""
        self.stopped_fixture()
        next(self.custody.glob('launchagent-*.plist')).write_bytes(b'CHANGED')
        with self.assertRaises(GUI.OWNER.Refused):
            GUI.reconcile(self.root, self.custody, resume=True)
        self.assertFalse(self.effects)

    def test_shutdown_deadline_preserves_stopped_phase(self):
        """A listener that never exits leaves explicit stopped custody and no bootstrap."""
        original = self.command

        def processes(args, **kwargs):
            """Keep a synthetic listener visible after stop without creating processes."""
            if args == ['ps', '-axo', 'comm='] and not self.is_loaded:
                return str(self.root / 'bin/Runner.Listener')
            return original(args, **kwargs)

        with mock.patch.object(GUI.OWNER, 'command', side_effect=processes):
            with self.assertRaises(GUI.OWNER.Refused):
                GUI.reconcile(self.root, self.custody, True)
        self.assertEqual([a[1] for a in self.effects], ['bootout'])
        self.assertEqual(json.loads((self.custody / 'session-change.json').read_text())['phase'], 'stopped')


    def test_resume_refuses_registration_drift_after_wait(self):
        """Registration changed during shutdown observation cannot inherit recovery authority."""
        self.stopped_fixture()

        def change_binding():
            """Alter only the synthetic work-folder binding before the final check."""
            (self.root / '.runner').write_text(json.dumps({**self.local, 'workFolder': 'OTHER'}))

        with mock.patch.object(GUI, 'wait_stopped', side_effect=change_binding):
            with self.assertRaises(GUI.OWNER.Refused):
                GUI.reconcile(self.root, self.custody, resume=True)
        self.assertFalse(self.effects)


    def test_exact_gui_absence_response(self):
        """Only the observed exact-domain not-found response is accepted as absence."""
        domain = 'gui/501'
        expected = 'Bad request.\nCould not find service "' + self.label + '" in domain for user gui: 501\n'
        with mock.patch.object(GUI.subprocess, 'run', return_value=mock.Mock(returncode=113, stderr=expected)) as run:
            self.assertTrue(self.gui_lookup_original(domain, self.label))
        self.assertEqual(run.call_args.args[0], ['/bin/launchctl', 'print', domain + '/' + self.label])
        self.assertEqual(run.call_args.kwargs['stdout'], GUI.subprocess.DEVNULL)

    def test_gui_lookup_errors_are_not_absence(self):
        """Permission errors, another GUI domain and unavailable lookups refuse recovery."""
        wrong = 'Bad request.\nCould not find service "' + self.label + '" in domain for user gui: 999\n'
        for code, stderr in [(1, 'permission denied'), (113, wrong), (113, 'OTHER')]:
            with self.subTest(code=code, stderr=stderr):
                with mock.patch.object(GUI.subprocess, 'run', return_value=mock.Mock(returncode=code, stderr=stderr)):
                    with self.assertRaises(GUI.OWNER.Refused):
                        self.gui_lookup_original('gui/501', self.label)
        with mock.patch.object(GUI.subprocess, 'run', side_effect=GUI.subprocess.TimeoutExpired('SYNTHETIC', 10)):
            with self.assertRaises(GUI.OWNER.Refused):
                self.gui_lookup_original('gui/501', self.label)

    def test_loaded_gui_not_overridden_by_caller_domain(self):
        """Caller-domain absence does not override exact GUI-domain loaded state."""
        self.stopped_fixture()
        self.is_loaded = True
        with mock.patch.object(GUI.OWNER, 'loaded', return_value=False) as caller:
            with self.assertRaises(GUI.OWNER.Refused):
                GUI.reconcile(self.root, self.custody, resume=True)
            caller.assert_not_called()
        self.assertFalse(self.effects)
        with mock.patch.object(GUI.subprocess, 'run', return_value=mock.Mock(returncode=0, stderr='')):
            self.assertFalse(self.gui_lookup_original('gui/501', self.label))


if __name__ == '__main__':
    unittest.main(verbosity=2)
