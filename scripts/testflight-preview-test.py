#!/usr/bin/env python3
"""Hermetic release controls; never access Apple, Keychain, real builds or uploads."""
import base64
import importlib.util
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import Mock, patch

spec = importlib.util.spec_from_file_location('preview', Path(__file__).with_name('testflight-preview.py'))
p = importlib.util.module_from_spec(spec)
spec.loader.exec_module(p)


class Response:
    """Finite fake Apple JSON response."""
    status = 200

    def __init__(self, value):
        """Serialize only synthetic fixture data."""
        self.raw = value if isinstance(value, bytes) else json.dumps(value).encode()

    def __enter__(self):
        """Enter synthetic HTTP response."""
        return self

    def __exit__(self, *args):
        """Keep test failures visible."""
        return False

    def read(self, bound):
        """Respect caller's finite output bound."""
        return self.raw[:bound]


class Tests(unittest.TestCase):
    """Check release selection, trust, transport, child custody and workflow shape."""

    def test_version_selection_and_controls(self):
        """Use higher published/beta train and globally higher numeric build."""
        self.assertEqual(p.select_numbers('0.1.0', ['1.0', '1.1.0'], ['5', '9.2']), ('1.1.1', '10'))
        self.assertEqual(p.select_numbers('2.0.0', ['1.0'], []), ('2.0.0', '1'))
        self.assertEqual(p.select_numbers('0.1.0', [], []), ('0.1.0', '1'))

    def test_version_refusals(self):
        """Refuse unsupported history and rollover rather than guessing."""
        for source, versions, builds in [('bad', [], []), ('1.0.0', ['1.0.99'], []),
                                         ('1.0.0', [], ['9999']), ('1.0.0', [], ['SECRET'])]:
            with self.subTest(source=source), self.assertRaises(p.Blocked):
                p.select_numbers(source, versions, builds)

    def test_exact_scoped_token(self):
        """JWT binds the actual fixed GET path without unscoped fallback."""
        asc = p.load_asc()
        with patch.object(asc, 'openssl', return_value=b'\x30\x06\x02\x01\x01\x02\x01\x02'):
            jwt = p.request_token(asc, 'ABCDEFGHIJ', '11111111-2222-3333-4444-555555555555', b'SYNTHETIC', '/v1/builds?filter%5Bapp%5D=123')
        encoded = jwt.split('.')[1]
        body = json.loads(base64.urlsafe_b64decode(encoded + '=' * (-len(encoded) % 4)))
        self.assertEqual(body['scope'], ['GET /v1/builds?filter%5Bapp%5D=123'])
        self.assertEqual(body['exp'] - body['iat'], 300)

    def test_api_transport_controls(self):
        """Use one fixed TLS origin, no proxies and exact path-scoped token."""
        asc = p.load_asc()
        opener = Mock()
        opener.open.return_value = Response({'data': [], 'links': {}})
        with patch.object(p, 'request_token', return_value='SYNTHETIC') as token, \
             patch.object(p.urllib.request, 'build_opener', return_value=opener) as factory:
            self.assertEqual(p.api(asc, 'kid', 'issuer', b'key', '/v1/apps?limit=2')['data'], [])
        token.assert_called_once_with(asc, 'kid', 'issuer', b'key', '/v1/apps?limit=2')
        request = opener.open.call_args.args[0]
        self.assertEqual(request.full_url, 'https://api.appstoreconnect.apple.com/v1/apps?limit=2')
        self.assertEqual(request.get_method(), 'GET')
        self.assertEqual(opener.open.call_count, 1)
        self.assertTrue(any(isinstance(x, asc.NoRedirect) for x in factory.call_args.args))
        proxy = next(x for x in factory.call_args.args if isinstance(x, p.urllib.request.ProxyHandler))
        self.assertEqual(proxy.proxies, {})

    def test_api_incomplete_or_oversized_refuses(self):
        """No partial inventory may select a build number."""
        asc = p.load_asc()
        for value in ({'data': [], 'links': {'next': 'PRIVATE'}}, {'links': []}, b'x' * 1048577):
            opener = Mock()
            opener.open.return_value = Response(value)
            with patch.object(p, 'request_token', return_value='SYNTHETIC'), \
                 patch.object(p.urllib.request, 'build_opener', return_value=opener), self.assertRaises(p.Blocked):
                p.api(asc, 'kid', 'issuer', b'key', '/v1/apps')
            opener.open.assert_called_once()

    def test_api_error_is_not_rendered(self):
        """Remote diagnostics and credentials never enter public errors."""
        opener = Mock()
        opener.open.side_effect = OSError('SECRET_REMOTE')
        with patch.object(p, 'request_token', return_value='SECRET_TOKEN'), \
             patch.object(p.urllib.request, 'build_opener', return_value=opener), self.assertRaises(p.Blocked) as error:
            p.api(p.load_asc(), 'kid', 'issuer', b'key', '/v1/apps')
        self.assertNotIn('SECRET', str(error.exception))

    def test_untrusted_context_never_reads_credentials(self):
        """Non-main context fails before credentials, effects or claiming resources."""
        with patch.dict(os.environ, {}, clear=True), patch.object(p, 'load_asc') as load, self.assertRaises(p.Blocked):
            p.release(True)
        load.assert_not_called()

    def test_child_failure_is_private_and_once(self):
        """Failed commands retain output locally without echoing it or retrying."""
        with tempfile.TemporaryDirectory() as directory:
            tools = p.Tools(Path(directory))
            with self.assertRaises(p.Blocked) as error:
                tools.call('synthetic', [sys.executable, '-c', 'print("SECRET_MARKER");raise SystemExit(3)'])
            self.assertNotIn('SECRET_MARKER', str(error.exception))
            self.assertIn('SECRET_MARKER', (Path(directory) / 'synthetic.log').read_text())
            self.assertEqual(len(list(Path(directory).iterdir())), 1)

    def test_child_timeout_is_stopped(self):
        """Bound and reap an owned sleeping subprocess without a retry."""
        with tempfile.TemporaryDirectory() as directory:
            tools = p.Tools(Path(directory))
            with self.assertRaises(subprocess.TimeoutExpired):
                tools.call('timeout', [sys.executable, '-c', 'import time;time.sleep(10)'], timeout=0.1)

    def test_log_cap_stops_child(self):
        """A noisy phase cannot fill runner storage past its finite log budget."""
        with tempfile.TemporaryDirectory() as directory:
            tools = p.Tools(Path(directory))
            with self.assertRaisesRegex(p.Blocked, 'output limit'):
                tools.call('noise', [sys.executable, '-c', 'print("x"*10000)'], log_limit=128)
            self.assertLessEqual((Path(directory) / 'noise.log').stat().st_size, 128)

    def test_exited_leader_descendant_is_stopped(self):
        """Reap an owned group even when its leader fails before the descendant."""
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            pidfile = root / 'child.pid'
            child = 'import signal,time;signal.signal(signal.SIGTERM,signal.SIG_IGN);time.sleep(30)'
            leader = ('import subprocess,sys,time;from pathlib import Path;'
                      'c=subprocess.Popen([sys.executable,"-c",sys.argv[2]]);'
                      'Path(sys.argv[1]).write_text(str(c.pid));time.sleep(.2);raise SystemExit(3)')
            with self.assertRaises(p.Blocked):
                p.Tools(root).call('leader', [sys.executable, '-c', leader, str(pidfile), child], timeout=8)
            pid = int(pidfile.read_text())
            state = subprocess.run(['ps', '-p', str(pid), '-o', 'state='], capture_output=True, text=True, timeout=5).stdout.strip()
            self.assertTrue(not state or state.startswith('Z'), state)

    def test_artifact_streamed_hash_and_symlink_refusal(self):
        """Hash exact regular bytes and refuse a linked or empty artifact."""
        import hashlib
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            ipa = root / 'app.ipa'
            ipa.write_bytes(b'x' * 2097153)
            self.assertEqual(p.artifact_digest(ipa), hashlib.sha256(b'x' * 2097153).hexdigest())
            alias = root / 'alias.ipa'
            alias.symlink_to(ipa)
            with self.assertRaises(OSError):
                p.artifact_digest(alias)
            ipa.write_bytes(b'')
            with self.assertRaises(p.Blocked):
                p.artifact_digest(ipa)

    def test_receipt_write_failure_still_releases_exact_lease(self):
        """Broken receipt storage cannot skip release after a stopped fake phase."""
        with tempfile.TemporaryDirectory() as directory:
            home = Path(directory)
            (home / 'Library/Caches').mkdir(parents=True)
            asc = Mock()
            asc.credentials.return_value = ('ABCDEFGHIJ', 'issuer', b'SYNTHETIC')
            asc.APP_REQUEST_PATH = '/v1/apps'
            values = [{'data': [{'id': '123', 'attributes': {'bundleId': p.BUNDLE}}]},
                      {'data': [], 'included': []}, {'data': []}]
            sha = 'a' * 40
            token = 'b' * 40
            fake = Mock()
            fake.call.side_effect = p.Blocked('stopped synthetic phase')
            claim = subprocess.CompletedProcess([], 0, 'WON: R-build-lease ' + token + '\n', '')
            release = subprocess.CompletedProcess([], 0, 'RELEASED: R-build-lease\n', '')
            with patch.dict(os.environ, {'GITHUB_REPOSITORY': 'narrate-it/prosody',
                 'GITHUB_REF': 'refs/heads/main', 'GITHUB_SHA': sha, 'RUNNER_OS': 'macOS', 'ASC_TEAM_ID': 'KLMNOPQRST'}, clear=True), \
                 patch.object(p.subprocess, 'check_output', side_effect=[sha + '\n', '']), \
                 patch.object(p, 'load_asc', return_value=asc), patch.object(p, 'api', side_effect=values), \
                 patch.object(p, 'signing_profile', return_value=('C' * 40, 'profile')), \
                 patch.object(p, 'capacity', return_value={}), patch.object(p.Path, 'home', return_value=home), \
                 patch.object(p, 'Tools', return_value=fake), \
                 patch.object(p.subprocess, 'run', side_effect=[claim, release]) as calls, \
                 patch.object(p, 'persist_receipt', side_effect=OSError('synthetic disk failure')), self.assertRaises(OSError):
                p.release(False)
            self.assertEqual(calls.call_count, 2)
            self.assertEqual(calls.call_args.args[0][-3:], ['release', 'R-build-lease', token])

    def test_receipt_failure_preserves_prior_and_never_uploads(self):
        """A failed fsync cannot truncate the prior receipt or enter upload."""
        with tempfile.TemporaryDirectory() as directory:
            folder = Path(directory)
            p.persist_receipt(folder, {'uploadState': 'UNKNOWN previous'})
            before = (folder / 'receipt.json').read_bytes()
            tools = Mock()
            with patch.object(p.os, 'fsync', side_effect=OSError('synthetic disk failure')), self.assertRaises(OSError):
                p.upload_once(tools, folder, {}, folder / 'app.ipa', [])
            tools.call.assert_not_called()
            self.assertEqual((folder / 'receipt.json').read_bytes(), before)

    def test_unknown_receipt_precedes_single_upload_and_survives_failure(self):
        """A failed external call keeps durable UNKNOWN and performs no retry."""
        with tempfile.TemporaryDirectory() as directory:
            folder = Path(directory)
            tools = Mock()
            def rejected(*args, **kwargs):
                """Check original uncertainty is persisted before transport stand-in."""
                value = json.loads((folder / 'receipt.json').read_text())
                self.assertTrue(value['uploadState'].startswith('UNKNOWN'))
                raise p.Blocked('synthetic lost response')
            tools.call.side_effect = rejected
            with self.assertRaises(p.Blocked):
                p.upload_once(tools, folder, {}, folder / 'app.ipa', [])
            tools.call.assert_called_once()
            self.assertTrue(json.loads((folder / 'receipt.json').read_text())['uploadState'].startswith('UNKNOWN'))

    def test_artifact_fifo_refuses_without_blocking(self):
        """Opening an IPA-shaped FIFO cannot wait for an external writer."""
        with tempfile.TemporaryDirectory() as directory:
            fifo = Path(directory) / 'app.ipa'
            os.mkfifo(fifo)
            with self.assertRaises(p.Blocked):
                p.artifact_digest(fifo)

    def test_profile_identity_and_denials(self):
        """Select matching store custody; refuse wrong team, expiry or development."""
        import copy
        import datetime
        import hashlib
        import plistlib
        cert = b'SYNTHETIC_CERTIFICATE'
        digest = hashlib.sha1(cert).hexdigest().upper()
        team = 'KLMNOPQRST'
        base = {'UUID': '11111111-2222-3333-4444-555555555555', 'TeamIdentifier': [team],
                'DeveloperCertificates': [cert], 'ExpirationDate': datetime.datetime(2099, 1, 1),
                'Entitlements': {'application-identifier': team + '.' + p.BUNDLE, 'get-task-allow': False}}
        with tempfile.TemporaryDirectory() as directory:
            home = Path(directory)
            profiles = home / 'Library/Developer/Xcode/UserData/Provisioning Profiles'
            profiles.mkdir(parents=True)
            (profiles / 'one.mobileprovision').touch()
            identity = '  1) ' + digest + ' "Apple Distribution: Example (' + team + ')"'
            for change in ['valid', 'team', 'expired', 'development', 'certificate', 'devices', 'enterprise']:
                value = copy.deepcopy(base)
                if change == 'team':
                    value['TeamIdentifier'] = ['OTHERTEAM1']
                elif change == 'expired':
                    value['ExpirationDate'] = datetime.datetime(2000, 1, 1)
                elif change == 'development':
                    value['Entitlements']['get-task-allow'] = True
                elif change == 'certificate':
                    value['DeveloperCertificates'] = [b'OTHER_CERTIFICATE']
                elif change == 'devices':
                    value['ProvisionedDevices'] = ['SYNTHETIC_DEVICE']
                elif change == 'enterprise':
                    value['ProvisionsAllDevices'] = True
                result = subprocess.CompletedProcess([], 0, plistlib.dumps(value), b'')
                with self.subTest(change=change), patch.object(p.Path, 'home', return_value=home), \
                     patch.object(p.subprocess, 'check_output', return_value=identity), \
                     patch.object(p.subprocess, 'run', return_value=result):
                    if change == 'valid':
                        self.assertEqual(p.signing_profile(team, home / 'keychain'), (digest, base['UUID']))
                    else:
                        with self.assertRaises(p.Blocked):
                            p.signing_profile(team, home / 'keychain')
            (profiles / 'two.mobileprovision').touch()
            result = subprocess.CompletedProcess([], 0, plistlib.dumps(base), b'')
            with patch.object(p.Path, 'home', return_value=home), \
                 patch.object(p.subprocess, 'check_output', return_value=identity), \
                 patch.object(p.subprocess, 'run', return_value=result), self.assertRaises(p.Blocked):
                p.signing_profile(team, home / 'keychain')

    def test_cleanup_failure_returns_without_unbounded_wait(self):
        """Keep failed cleanup observable without waiting for the live child exit."""
        import time
        children = []
        real_popen = subprocess.Popen
        def start(*args, **kwargs):
            """Retain exact synthetic child custody for the test's final cleanup."""
            child = real_popen(*args, **kwargs)
            children.append(child)
            return child
        try:
            with tempfile.TemporaryDirectory() as directory:
                tools = p.Tools(Path(directory))
                before = time.monotonic()
                with patch.object(p.subprocess, 'Popen', side_effect=start), \
                     patch.object(p, 'stop_group', side_effect=p.Blocked('synthetic cleanup failure')), \
                     self.assertRaisesRegex(p.Blocked, 'synthetic cleanup failure'):
                    tools.call('cleanup-failure', [sys.executable, '-c', 'import time;time.sleep(8)'], timeout=0.1)
                self.assertLess(time.monotonic() - before, 2)
                self.assertTrue(tools.cleanup_unconfirmed)
                self.assertEqual(tools.unconfirmed_groups, [children[0].pid])
        finally:
            for child in children:
                try:
                    os.killpg(child.pid, p.signal.SIGKILL)
                except ProcessLookupError:
                    pass
                child.wait(timeout=5)

    def test_capacity_finite_samples_and_refusals(self):
        """Require real interval idle, pressure, thermal and sustained-swap bounds."""
        vm = 'Mach Virtual Memory Statistics: (page size of 4096 bytes)\nSwapins: {swap}\nSwapouts: 0\n'
        normal = 'No thermal warning level has been recorded\nNo performance warning level has been recorded'
        for case in ['valid', 'idle', 'pressure', 'swap', 'thermal', 'missing-load']:
            idle = [10, 70, 80] if case != 'idle' else [99, 5, 10]
            raw = ''.join(('Load Avg: 3, 2, 1\n' if case != 'missing-load' else '') +
                          'CPU usage: 1% user, 1% sys, ' + str(n) + '% idle\n' for n in idle)
            swaps = [0, 2000, 4000] if case == 'swap' else [0, 0, 0]
            values = []
            for i in range(3):
                values.extend([vm.format(swap=swaps[i]), '2' if case == 'pressure' and i == 1 else '1'])
            values.append('unknown' if case == 'thermal' else normal)
            child = Mock()
            child.communicate.return_value = (raw, None)
            child.poll.return_value = 0
            with self.subTest(case=case), patch.object(p.subprocess, 'Popen', return_value=child), \
                 patch.object(p.subprocess, 'check_output', side_effect=values), \
                 patch.object(p.time, 'sleep'), patch.object(p.time, 'monotonic', side_effect=[0, 5, 10]):
                if case == 'valid':
                    self.assertEqual(p.capacity()['intervalIdle'], [70, 80])
                else:
                    with self.assertRaises(p.Blocked):
                        p.capacity()
            child.communicate.assert_called_once_with(timeout=20)
            child.stdout.close.assert_called_once()

    def test_archive_profile_is_not_a_global_override(self):
        """Actual archive argv must not force app profiles onto dependency targets."""
        command = p.archive_command('KLMNOPQRST', 'C' * 40, 'SYNTHETIC_PROFILE', Path('/tmp/keychain'), Path('/tmp/archive'))
        self.assertIn('PROSODY_APP_PROFILE=SYNTHETIC_PROFILE', command)
        self.assertFalse(any(x.startswith('PROVISIONING_PROFILE_SPECIFIER=') for x in command))
        self.assertIn('CODE_SIGN_STYLE=Manual', command)
        self.assertEqual(command[-1], 'archive')

    def test_profile_override_belongs_only_to_runner_release(self):
        """Parse the real Xcode project and enforce app-only profile inheritance."""
        path = p.ROOT / 'mobile/ios/Runner.xcodeproj/project.pbxproj'
        objects = json.loads(subprocess.check_output(['/usr/bin/plutil', '-convert', 'json', '-o', '-', str(path)], timeout=10))['objects']
        targets = [x for x in objects.values() if x.get('isa') == 'PBXNativeTarget' and x.get('name') == 'Runner']
        self.assertEqual(len(targets), 1)
        ids = objects[targets[0]['buildConfigurationList']]['buildConfigurations']
        release = [i for i in ids if objects[i].get('name') == 'Release']
        self.assertEqual(len(release), 1)
        self.assertEqual(objects[release[0]]['buildSettings'].get('PROVISIONING_PROFILE_SPECIFIER'), '$(PROSODY_APP_PROFILE)')
        for key, value in objects.items():
            if key != release[0] and value.get('isa') == 'XCBuildConfiguration':
                self.assertNotIn('PROSODY_APP_PROFILE', json.dumps(value))

    def assert_workflow(self, value):
        """Validate actual parsed workflow security and explicit upload selection."""
        self.assertEqual(set(value['on']), {'workflow_dispatch'})
        self.assertFalse(value['on']['workflow_dispatch']['inputs']['upload']['default'])
        self.assertEqual(value['permissions'], {'contents': 'read'})
        self.assertFalse(value['concurrency']['cancel-in-progress'])
        self.assertEqual(set(value['jobs']), {'preview'})
        j = value['jobs']['preview']
        self.assertEqual(j['environment'], 'prosody-asc')
        self.assertEqual(j['runs-on'], ['self-hosted', 'macOS', 'prosody'])
        self.assertEqual(j['if'], "github.repository == 'narrate-it/prosody' && github.ref == 'refs/heads/main'")
        self.assertNotIn('env', value)
        self.assertNotIn('env', j)
        self.assertEqual(j['steps'][0]['with'], {'ref': '${{ github.sha }}', 'persist-credentials': False, 'clean': False})
        self.assertEqual(j['steps'][0]['uses'], 'actions/checkout@11d5960a326750d5838078e36cf38b85af677262')
        self.assertNotIn('env', j['steps'][0])
        env = j['steps'][1]['env']
        for name in ['ASC_API_KEY', 'ASC_KEY_ID', 'ASC_ISSUER_ID', 'ASC_TEAM_ID']:
            self.assertEqual(env[name], '${{ secrets.' + name + ' }}')
        self.assertIn('"$PREVIEW_UPLOAD" = "true"', j['steps'][1]['run'])
        self.assertIn('/usr/bin/python3 -I scripts/testflight-preview.py --upload', j['steps'][1]['run'])

    def test_workflow_and_mutation_controls(self):
        """Reject moved trust/secret/runner controls in parsed JSON-form YAML."""
        import copy
        value = json.loads((p.ROOT / '.github/workflows/testflight-preview.yml').read_text())
        self.assert_workflow(value)
        for mutation in ['environment', 'runner', 'main', 'secrets', 'upload-default']:
            changed = copy.deepcopy(value)
            j = changed['jobs']['preview']
            if mutation == 'environment':
                j.pop('environment')
            elif mutation == 'runner':
                j['runs-on'] = ['macos-latest']
            elif mutation == 'main':
                j['if'] = 'true'
            elif mutation == 'secrets':
                changed['env'] = j['steps'][1].pop('env')
            else:
                changed['on']['workflow_dispatch']['inputs']['upload']['default'] = True
            with self.subTest(mutation=mutation), self.assertRaises((AssertionError, KeyError)):
                self.assert_workflow(changed)


if __name__ == '__main__':
    unittest.main(verbosity=2)
