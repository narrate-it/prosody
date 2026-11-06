#!/usr/bin/env python3
"""Hermetic ASC preflight controls; no Apple request or real credential use."""
import base64
import contextlib
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
        self.body = body

    def __enter__(self):
        return self

    def __exit__(self, *_):
        return False

    def read(self, maximum):
        return self.body[:maximum]


class Tests(unittest.TestCase):
    """Exercise refusal, transport, signature and safe-output boundaries."""

    def test_credentials_consumed_raw_and_base64(self):
        key = b'-----BEGIN PRIVATE KEY-----\nSYNTHETIC\n-----END PRIVATE KEY-----\n'
        for value in (key.decode(), base64.b64encode(key).decode()):
            env = {'ASC_API_KEY': value, 'ASC_KEY_ID': 'ABCDEFGHIJ', 'ASC_TEAM_ID': 'KLMNOPQRST',
                   'ASC_ISSUER_ID': '11111111-2222-3333-4444-555555555555', 'KEEP': 'yes'}
            self.assertEqual(asc.credentials(env)[2].rstrip(), key.rstrip())
            self.assertEqual(env, {'KEEP': 'yes'})

    def test_bad_preflight_never_transports_or_renders_secret(self):
        with patch.dict(os.environ, {'ASC_API_KEY': 'SECRET_MARKER'}, clear=True), \
             patch.object(asc, 'read_app') as read, contextlib.redirect_stdout(io.StringIO()) as output:
            self.assertEqual(asc.main(), 1)
        read.assert_not_called()
        self.assertNotIn('SECRET_MARKER', output.getvalue())

    def test_der_bounds(self):
        valid = b'\x30\x06\x02\x01\x01\x02\x01\x02'
        self.assertEqual(asc.raw_signature(valid), (b'\0' * 31 + b'\1') + (b'\0' * 31 + b'\2'))
        for value in (valid + b'x', valid[:-1], b'\x30\x06\x02\x01\x80\x02\x01\x02',
                      b'\x30\x07\x02\x02\x00\x01\x02\x01\x02',
                      b'\x30\x06\x02\x01\x00\x02\x01\x02'):
            with self.subTest(value=value), self.assertRaises(asc.Refused):
                asc.raw_signature(value)

    def test_real_synthetic_signature_and_cleanup(self):
        # Real system signer; ephemeral synthetic P-256 key, never Apple credentials.
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
        with self.assertRaises(asc.Refused):
            asc.NoRedirect().redirect_request(None, None, 302, '', {}, 'https://evil.invalid/')

    def inspect(self, value):
        opener = unittest.mock.Mock()
        opener.open.return_value = Response(value if isinstance(value, bytes) else json.dumps(value).encode())
        with patch.object(asc.urllib.request, 'build_opener', return_value=opener):
            asc.read_app('SYNTHETIC_TOKEN')
        request = opener.open.call_args.args[0]
        self.assertEqual(request.get_method(), 'GET')
        self.assertTrue(request.full_url.startswith(asc.ORIGIN + '/v1/apps?'))
        self.assertEqual(asc.urllib.parse.parse_qs(asc.urllib.parse.urlsplit(request.full_url).query)
                         ['filter[bundleId]'], [asc.BUNDLE])
        self.assertEqual(opener.open.call_args.kwargs['timeout'], 15)

    def test_exact_existing_app(self):
        self.inspect({'data': [{'type': 'apps', 'id': '123456', 'attributes': {'bundleId': asc.BUNDLE}}],
                      'links': {'next': None}})

    def test_missing_ambiguous_foreign_and_oversized_refused(self):
        app = {'type': 'apps', 'id': '123', 'attributes': {'bundleId': asc.BUNDLE}}
        foreign = {'type': 'apps', 'id': '123', 'attributes': {'bundleId': 'other.app'}}
        for value in ({'data': []}, {'data': [app, app]}, {'data': [foreign]},
                      {'data': [app], 'links': {'next': 'unexpected'}}, b'x' * 65537, b'not JSON'):
            with self.subTest(value=str(value)[:50]), self.assertRaises(asc.Refused):
                self.inspect(value)

    def test_remote_error_not_rendered(self):
        opener = unittest.mock.Mock()
        opener.open.side_effect = urllib.error.HTTPError(asc.ORIGIN, 403, 'SECRET_REMOTE_BODY', {}, io.BytesIO(b'PRIVATE'))
        with patch.object(asc.urllib.request, 'build_opener', return_value=opener), \
             patch.object(asc, 'credentials', return_value=('id', 'issuer', b'key')), \
             patch.object(asc, 'token', return_value='SECRET_TOKEN'), \
             contextlib.redirect_stdout(io.StringIO()) as output:
            self.assertEqual(asc.main(), 1)
        self.assertNotIn('SECRET', output.getvalue())
        self.assertNotIn('PRIVATE', output.getvalue())

    def test_workflow_manual_trusted_main_and_selfhosted(self):
        text = (Path(__file__).parent.parent / '.github/workflows/asc-preflight.yml').read_text()
        self.assertIn('workflow_dispatch:', text)
        self.assertNotIn('pull_request', text)
        self.assertNotIn('macos-latest', text)
        self.assertIn('runs-on: [self-hosted, macOS, prosody]', text)
        self.assertIn("github.ref == 'refs/heads/main'", text)
        self.assertIn('persist-credentials: false', text)
        self.assertIn('run: /usr/bin/python3 -I scripts/asc-preflight.py', text)

    def test_isolation_refuses_retained_module_and_pythonpath(self):
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
