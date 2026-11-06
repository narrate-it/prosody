#!/usr/bin/env python3
"""Hermetic ASC preflight controls; no Apple request or real credential use."""
import base64
import contextlib
import copy
import importlib.util
import io
import json
import os
import selectors
import sys
import subprocess
import tempfile
import unittest
import urllib.error
from pathlib import Path
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('asc', Path(__file__).with_name('asc-preflight.py'))
asc = importlib.util.module_from_spec(spec)
spec.loader.exec_module(asc)


class Response:
    """A bounded fake Apple response."""
    status = 200

    def __init__(self, body):
        """Retain the synthetic response bytes."""
        self.body = body

    def __enter__(self):
        """Expose this fake as a context-managed response."""
        return self

    def __exit__(self, *_):
        """Preserve test exceptions from the fake context."""
        return False

    def read(self, maximum):
        """Return at most the requested response bytes."""
        return self.body[:maximum]


class Tests(unittest.TestCase):
    """Exercise refusal, transport, signature and safe-output boundaries."""

    def test_credentials_consumed_raw_and_base64(self):
        """Consume only configured secret entries in both supported encodings."""
        key = b'-----BEGIN PRIVATE KEY-----\nSYNTHETIC\n-----END PRIVATE KEY-----\n'
        for value in (key.decode(), base64.b64encode(key).decode()):
            env = {'ASC_API_KEY': value, 'ASC_KEY_ID': 'ABCDEFGHIJ', 'ASC_TEAM_ID': 'KLMNOPQRST',
                   'ASC_ISSUER_ID': '11111111-2222-3333-4444-555555555555', 'KEEP': 'yes'}
            self.assertEqual(asc.credentials(env)[2].rstrip(), key.rstrip())
            self.assertEqual(env, {'KEEP': 'yes'})

    def test_bad_preflight_never_transports_or_renders_secret(self):
        """Reject missing credentials before transport without echoing them."""
        with patch.dict(os.environ, {'ASC_API_KEY': 'SECRET_MARKER'}, clear=True), \
             patch.object(asc, 'read_app') as read, contextlib.redirect_stdout(io.StringIO()) as output:
            self.assertEqual(asc.main(), 1)
        read.assert_not_called()
        self.assertNotIn('SECRET_MARKER', output.getvalue())

    def test_der_bounds(self):
        """Accept canonical P-256 scalars and reject malformed DER."""
        valid = b'\x30\x06\x02\x01\x01\x02\x01\x02'
        self.assertEqual(asc.raw_signature(valid), (b'\0' * 31 + b'\1') + (b'\0' * 31 + b'\2'))
        for value in (valid + b'x', valid[:-1], b'\x30\x06\x02\x01\x80\x02\x01\x02',
                      b'\x30\x07\x02\x02\x00\x01\x02\x01\x02',
                      b'\x30\x06\x02\x01\x00\x02\x01\x02'):
            with self.subTest(value=value), self.assertRaises(asc.Refused):
                asc.raw_signature(value)

    def test_real_synthetic_signature_and_cleanup(self):
        # Real system signer; ephemeral synthetic P-256 key, never Apple credentials.
        """Verify a synthetic ES256 token using the real installed signer."""
        key = subprocess.run(['/usr/bin/openssl', 'genpkey', '-algorithm', 'EC', '-pkeyopt',
                              'ec_paramgen_curve:prime256v1', '-pkeyopt', 'ec_param_enc:named_curve'], capture_output=True, check=True, timeout=10).stdout
        with patch.object(asc.time, 'time', return_value=1000):
            jwt = asc.token('ABCDEFGHIJ', '11111111-2222-3333-4444-555555555555', key)
        header, claims, encoded = jwt.split('.')
        decode = lambda value: base64.urlsafe_b64decode(value + '=' * (-len(value) % 4))
        self.assertEqual(json.loads(decode(header))['alg'], 'ES256')
        payload = json.loads(decode(claims))
        self.assertEqual(payload['scope'], ['GET /v1/apps'])
        self.assertEqual(payload['exp'] - payload['iat'], 300)
        raw = decode(encoded)
        self.assertEqual(len(raw), 64)
        parts = []
        for value in (raw[:32], raw[32:]):
            value = value.lstrip(b'\0')
            if value[0] & 128:
                value = b'\0' + value
            parts.append(b'\x02' + bytes([len(value)]) + value)
        der = b''.join(parts)
        der = b'\x30' + bytes([len(der)]) + der
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'key').write_bytes(key)
            public = subprocess.run(['/usr/bin/openssl', 'pkey', '-in', str(root / 'key'), '-pubout'],
                                    capture_output=True, check=True, timeout=10).stdout
            (root / 'public').write_bytes(public)
            (root / 'sig').write_bytes(der)
            verified = subprocess.run(['/usr/bin/openssl', 'dgst', '-sha256', '-verify', str(root / 'public'),
                                       '-signature', str(root / 'sig')], input=(header + '.' + claims).encode(),
                                      capture_output=True, timeout=10)
            self.assertEqual(verified.returncode, 0)

    def test_redirect_refused(self):
        """Never follow a redirect with the bearer credential."""
        with self.assertRaises(asc.Refused):
            asc.NoRedirect().redirect_request(None, None, 302, '', {}, 'https://evil.invalid/')

    def inspect(self, value):
        """Run one fake response and verify the exact GET identity and timeout."""
        opener = unittest.mock.Mock()
        opener.open.return_value = Response(value if isinstance(value, bytes) else json.dumps(value).encode())
        with patch.object(asc.urllib.request, 'build_opener', return_value=opener) as build:
            asc.read_app('SYNTHETIC_TOKEN')
        handlers = build.call_args.args
        self.assertEqual(len(handlers), 3)
        self.assertTrue(any(isinstance(handler, asc.NoRedirect) for handler in handlers))
        proxy = next(handler for handler in handlers if isinstance(handler, asc.urllib.request.ProxyHandler))
        self.assertEqual(proxy.proxies, {})
        self.assertTrue(any(isinstance(handler, asc.urllib.request.HTTPSHandler) for handler in handlers))
        request = opener.open.call_args.args[0]
        self.assertEqual(request.get_method(), 'GET')
        self.assertTrue(request.full_url.startswith(asc.ORIGIN + '/v1/apps?'))
        self.assertEqual(asc.urllib.parse.parse_qs(asc.urllib.parse.urlsplit(request.full_url).query)
                         ['filter[bundleId]'], [asc.BUNDLE])
        self.assertEqual(opener.open.call_args.kwargs['timeout'], 15)
        self.assertEqual(request.get_header('Authorization'), 'Bearer SYNTHETIC_TOKEN')
        query = asc.urllib.parse.parse_qs(asc.urllib.parse.urlsplit(request.full_url).query)
        self.assertEqual(query['limit'], ['2'])
        self.assertEqual(query['fields[apps]'], ['bundleId'])

    def test_exact_existing_app(self):
        """Accept one exact existing bundle match."""
        self.inspect({'data': [{'type': 'apps', 'id': '123456', 'attributes': {'bundleId': asc.BUNDLE}}],
                      'links': {'next': None}})

    def test_missing_ambiguous_foreign_and_oversized_refused(self):
        """Refuse incomplete, foreign, malformed and oversized API results."""
        app = {'type': 'apps', 'id': '123', 'attributes': {'bundleId': asc.BUNDLE}}
        foreign = {'type': 'apps', 'id': '123', 'attributes': {'bundleId': 'other.app'}}
        for value in ({'data': []}, {'data': [app, app]}, {'data': [foreign]},
                      {'data': [app], 'links': {'next': 'unexpected'}}, b'x' * 65537, b'not JSON'):
            with self.subTest(value=str(value)[:50]), self.assertRaises(asc.Refused):
                self.inspect(value)

    def test_malformed_links_have_safe_refusal(self):
        """Null/string links use a fixed refusal without remote data or credentials."""
        app = {'type': 'apps', 'id': '123', 'attributes': {'bundleId': asc.BUNDLE}}
        for links in (None, 'SECRET_REMOTE_BODY'):
            opener = unittest.mock.Mock()
            opener.open.return_value = Response(json.dumps({'data': [app], 'links': links}).encode())
            with self.subTest(links_type=type(links).__name__), \
                 patch.object(asc.urllib.request, 'build_opener', return_value=opener), \
                 patch.object(asc, 'credentials', return_value=('id', 'issuer', b'key')), \
                 patch.object(asc, 'token', return_value='SECRET_TOKEN'), \
                 contextlib.redirect_stdout(io.StringIO()) as output:
                self.assertEqual(asc.main(), 1)
            self.assertTrue(output.getvalue().startswith('BLOCKED: '))
            self.assertNotIn('SECRET', output.getvalue())

    def test_remote_error_not_rendered(self):
        """Expose a fixed refusal without remote or token material."""
        opener = unittest.mock.Mock()
        opener.open.side_effect = urllib.error.HTTPError(asc.ORIGIN, 403, 'SECRET_REMOTE_BODY', {}, io.BytesIO(b'PRIVATE'))
        with patch.object(asc.urllib.request, 'build_opener', return_value=opener), \
             patch.object(asc, 'credentials', return_value=('id', 'issuer', b'key')), \
             patch.object(asc, 'token', return_value='SECRET_TOKEN'), \
             contextlib.redirect_stdout(io.StringIO()) as output:
            self.assertEqual(asc.main(), 1)
        self.assertNotIn('SECRET', output.getvalue())
        self.assertNotIn('PRIVATE', output.getvalue())

    def assert_workflow(self, document):
        """Check actual job/step boundaries, never text that could be a comment."""
        self.assertEqual(document['on'], {'workflow_dispatch': None})
        self.assertEqual(document['permissions'], {'contents': 'read'})
        self.assertNotIn('env', document)
        self.assertEqual(set(document['jobs']), {'read-existing-app'})
        job = document['jobs']['read-existing-app']
        self.assertEqual(job['if'], "github.repository == 'narrate-it/prosody' && github.ref == 'refs/heads/main'")
        self.assertEqual(job['runs-on'], ['self-hosted', 'macOS', 'prosody'])
        self.assertEqual(job['timeout-minutes'], 3)
        self.assertEqual(job['environment'], 'prosody-asc')
        self.assertNotIn('env', job)
        self.assertNotIn('permissions', job)
        checkout, read = job['steps']
        self.assertEqual(checkout['uses'], 'actions/checkout@11d5960a326750d5838078e36cf38b85af677262')
        self.assertEqual(checkout['with'], {'ref': '${{ github.sha }}', 'persist-credentials': False, 'clean': False})
        self.assertNotIn('env', checkout)
        self.assertEqual(read['run'], '/usr/bin/python3 -I scripts/asc-preflight.py')
        self.assertEqual(read['env'], {key: '${{ secrets.PROSODY_' + key + ' }}' for key in
                                     ['ASC_API_KEY', 'ASC_ISSUER_ID', 'ASC_KEY_ID', 'ASC_TEAM_ID']})

    def test_workflow_manual_trusted_main_and_selfhosted(self):
        """Parse the JSON-form YAML and verify trusted, read-only secret scope."""
        workflow = Path(__file__).parent.parent / '.github/workflows/asc-preflight.yml'
        self.assert_workflow(json.loads(workflow.read_text()))

    def test_workflow_misplaced_guards_and_secrets_fail(self):
        """Valid-looking text in another field cannot replace security structure."""
        workflow = Path(__file__).parent.parent / '.github/workflows/asc-preflight.yml'
        original = json.loads(workflow.read_text())
        for kind in ('guard', 'credentials', 'permission', 'global-secret', 'checkout-secret', 'environment', 'repository-fallback'):
            with self.subTest(kind=kind):
                value = copy.deepcopy(original)
                job = value['jobs']['read-existing-app']
                if kind == 'guard':
                    value['name'] = job.pop('if')
                elif kind == 'credentials':
                    job['steps'][1]['name'] = str(job['steps'][0]['with'].pop('persist-credentials'))
                elif kind == 'permission':
                    value['permissions']['contents'] = 'write'
                elif kind == 'global-secret':
                    value['env'] = job['steps'][1].pop('env')
                elif kind == 'environment':
                    job.pop('environment')
                elif kind == 'repository-fallback':
                    job['steps'][1]['env']['ASC_API_KEY'] = '${{ secrets.ASC_API_KEY }}'
                else:
                    job['steps'][0]['env'] = job['steps'][1].pop('env')
                with self.assertRaises((AssertionError, KeyError)):
                    self.assert_workflow(value)

    def test_isolation_refuses_retained_module_and_pythonpath(self):
        """Prevent retained startup modules from executing in the isolated interpreter."""
        script = Path(__file__).with_name('asc-preflight.py').resolve()
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'json.py').write_text("raise RuntimeError('SHADOW_EXECUTED')")
            (root / 'sitecustomize.py').write_text("raise RuntimeError('SHADOW_EXECUTED')")
            environment = dict(os.environ, PYTHONPATH=directory, ASC_API_KEY='SYNTHETIC')
            result = subprocess.run(['/usr/bin/python3', '-I', str(script)], cwd=root,
                                    env=environment, capture_output=True, timeout=10)
            self.assertEqual(result.returncode, 1)
            self.assertNotIn(b'SHADOW_EXECUTED', result.stdout + result.stderr)
            self.assertIn(b'BLOCKED:', result.stdout)

    def test_sigterm_unwinds_private_key(self):
        """Confirm normal termination removes the synthetic temporary key."""
        script = Path(__file__).with_name('asc-preflight.py').resolve()
        code = """import importlib.util, sys, time
spec = importlib.util.spec_from_file_location('asc', sys.argv[1])
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)
def block(args, data=None):
    print(args[args.index('-in') + 1], flush=True)
    time.sleep(60)
m.openssl = block
sys.exit(m.main())
"""
        environment = dict(os.environ, ASC_API_KEY='-----BEGIN PRIVATE KEY-----\nSYNTHETIC\n-----END PRIVATE KEY-----',
                           ASC_KEY_ID='ABCDEFGHIJ', ASC_TEAM_ID='KLMNOPQRST',
                           ASC_ISSUER_ID='11111111-2222-3333-4444-555555555555')
        child = subprocess.Popen(['/usr/bin/python3', '-I', '-c', code, str(script)],
                                 env=environment, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        try:
            with selectors.DefaultSelector() as selector:
                selector.register(child.stdout, selectors.EVENT_READ)
                self.assertTrue(selector.select(10), 'synthetic signer never started')
            path = Path(child.stdout.readline().decode().strip())
            self.assertTrue(path.is_file())
            self.assertEqual(path.stat().st_mode & 0o777, 0o600)
            child.terminate()
            output, errors = child.communicate(timeout=10)
            self.assertEqual(child.returncode, 1)
            self.assertIn(b'BLOCKED: Preflight terminated', output)
            self.assertFalse(path.parent.exists())
            self.assertEqual(errors, b'')
        finally:
            if child.poll() is None:
                child.kill()
                child.communicate(timeout=5)


if __name__ == '__main__':
    unittest.main(verbosity=2)
