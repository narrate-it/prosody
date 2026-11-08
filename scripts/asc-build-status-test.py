#!/usr/bin/env python3
"""Hermetic exact-build reader tests; never use Apple credentials or network."""
import base64
import contextlib
import copy
import importlib.util
import io
import json
import os
import unittest
import urllib.error
from pathlib import Path
from unittest.mock import Mock, patch

spec = importlib.util.spec_from_file_location('asc', Path(__file__).with_name('asc-preflight.py'))
asc = importlib.util.module_from_spec(spec)
spec.loader.exec_module(asc)


def document():
    """Return a synthetic exact app/train/build association."""
    return {'data': [{'type': 'builds', 'id': 'build-id',
                     'attributes': {'version': '41', 'processingState': 'VALID', 'expired': False},
                     'relationships': {'app': {'data': {'type': 'apps', 'id': '123'}},
                                       'preReleaseVersion': {'data': {'type': 'preReleaseVersions', 'id': 'train-id'}}}}],
            'included': [{'type': 'preReleaseVersions', 'id': 'train-id',
                          'attributes': {'version': '1.0.1', 'platform': 'IOS'}}]}


class Response:
    """Bounded fake transport response with no external effects."""
    status = 200

    def __init__(self, raw):
        """Retain exact fake response bytes."""
        self.raw = raw

    def __enter__(self):
        """Expose the context-managed response."""
        return self

    def __exit__(self, *_):
        """Do not suppress validation exceptions."""
        return False

    def read(self, size):
        """Respect the reader's requested byte bound."""
        return self.raw[:size]


class Tests(unittest.TestCase):
    """Check exact identity, least privilege, safe diagnostics and workflow trust."""

    def read(self, value):
        """Invoke the real validator through a fake bounded body read."""
        raw = value if isinstance(value, bytes) else json.dumps(value).encode()
        with patch.object(asc, 'read_response', return_value=raw) as read:
            result = asc.read_build('SYNTHETIC_TOKEN', '123', '1.0.1', '41')
        read.assert_called_once_with('SYNTHETIC_TOKEN', asc.build_request_path('123', '1.0.1', '41'))
        return result

    def test_valid_exact_build(self):
        """Accept a correctly bound build and expose only finite status flags."""
        self.assertEqual(self.read(document()), ('VALID', False))

    def test_all_known_states_and_expiry(self):
        """Report observations without turning successful reads into tester readiness."""
        for state in ['PROCESSING', 'FAILED', 'INVALID', 'VALID']:
            for expired in [False, True]:
                value = document()
                value['data'][0]['attributes'].update(processingState=state, expired=expired)
                self.assertEqual(self.read(value), (state, expired))

    def test_cross_app_build_refused(self):
        """A matching version/build from another app must fail closed."""
        value = document()
        value['data'][0]['relationships']['app']['data']['id'] = '999'
        with self.assertRaises(asc.Refused):
            self.read(value)

    def test_train_and_build_mismatches_refused(self):
        """Reject wrong platform, train, relationship, version and resource types."""
        changes = [
            lambda d: d['data'][0]['attributes'].update(version='42'),
            lambda d: d['included'][0]['attributes'].update(version='1.0.2'),
            lambda d: d['included'][0]['attributes'].update(platform='MAC_OS'),
            lambda d: d['included'][0].update(id='another-train'),
            lambda d: d['data'][0].update(type='apps'),
            lambda d: d['included'][0].update(type='apps'),
            lambda d: d['data'][0].update(id='private\nidentifier'),
        ]
        for change in changes:
            value = document()
            change(value)
            with self.assertRaises(asc.Refused):
                self.read(value)

    def test_missing_ambiguous_or_paginated_refused(self):
        """Never select the first of ambiguous rows or follow pagination links."""
        changes = [lambda d: d.update(data=[]),
                   lambda d: d['data'].append(copy.deepcopy(d['data'][0])),
                   lambda d: d.update(included=[]),
                   lambda d: d['included'].append(copy.deepcopy(d['included'][0])),
                   lambda d: d.update(links={'next': 'https://private.invalid/next'}),
                   lambda d: d.update(links=[]),
                   lambda d: d['data'][0]['relationships'].pop('app')]
        for change in changes:
            value = document()
            change(value)
            with self.assertRaises(asc.Refused):
                self.read(value)

    def test_bad_status_or_expiry_refused(self):
        """Reject missing, coercible and unrecognized status data."""
        for key, bad in [('processingState', 'PRIVATE_STATUS'), ('processingState', None),
                         ('expired', 'false'), ('expired', 0), ('expired', None)]:
            value = document()
            value['data'][0]['attributes'][key] = bad
            with self.assertRaises(asc.Refused):
                self.read(value)

    def test_duplicate_utf8_bom_and_non_json_refused(self):
        """Refuse erased duplicate keys, invalid UTF-8, BOM and nonfinite JSON."""
        raw = json.dumps(document()).encode()
        bodies = [raw.replace(b'"expired": false', b'"expired": false, "expired": true'),
                  raw.replace(b'"expired": false', b'"expired": false, "expir\\u0065d": true'),
                  b'\xef\xbb\xbf' + raw, b'\xff' + raw, b'private non-json',
                  raw[:-1] + b', "ignored": NaN}']
        for body in bodies:
            with self.assertRaises(asc.Refused):
                self.read(body)

    def test_positive_float_overflow_refused(self):
        """A valid numeric lexeme must not become accepted positive infinity."""
        raw = json.dumps(document()).encode()[:-1] + b', "ignored": 1e999}'
        with self.assertRaises(asc.Refused):
            self.read(raw)

    def test_negative_float_overflow_refused(self):
        """A valid numeric lexeme must not become accepted negative infinity."""
        raw = json.dumps(document()).encode()[:-1] + b', "ignored": -1e999}'
        with self.assertRaises(asc.Refused):
            self.read(raw)

    def test_finite_ignored_number_preserved(self):
        """Finite unselected metadata does not alter the selected status flags."""
        value = document()
        value['ignored'] = 1.5
        self.assertEqual(self.read(value), ('VALID', False))

    def test_selection_and_path_are_closed(self):
        """Allow only canonical numeric selectors and exact profile queries."""
        self.assertIsNone(asc.selection([]))
        self.assertEqual(asc.selection(['--build-version', '1.0.1', '--build-number', '41']), ('1.0.1', '41'))
        for args in [['SECRET'], ['--build-version', '01.0.1', '--build-number', '41'],
                     ['--build-version', '1.0.1', '--build-number', '0'],
                     ['--build-version', '1.0.1&secret=x', '--build-number', '41']]:
            with self.assertRaises(asc.Refused):
                asc.selection(args)
        path = asc.build_request_path('123', '1.0.1', '41')
        self.assertEqual(asc.request_scope(path), path)
        self.assertEqual(asc.request_scope(asc.APP_REQUEST_PATH), asc.APP_REQUEST_PATH)
        for bad in ['/v1/builds', '/v1/apps', 'https://private.invalid' + path,
                    path + '&extra=secret', path + '#fragment', path.replace('limit=2', 'limit=200'),
                    path + '&filter%5Bapp%5D=999']:
            with patch.object(asc, 'openssl') as signer, self.assertRaises(asc.Refused):
                asc.token('ABCDEFGHIJ', 'issuer', b'SYNTHETIC', bad)
            signer.assert_not_called()

    def test_jwt_scope_matches_filtered_request(self):
        """Bind the build JWT to one GET and the exact canonical encoded query."""
        path = asc.build_request_path('123', '1.0.1', '41')
        prefix = bytes.fromhex('3059301306072a8648ce3d020106082a8648ce3d03010703420004')
        with patch.object(asc, 'openssl', side_effect=[prefix.ljust(91, b'\0'), b'\x30\x06\x02\x01\x01\x02\x01\x02']):
            jwt = asc.token('ABCDEFGHIJ', 'issuer', b'SYNTHETIC', path)
        segment = jwt.split('.')[1]
        claims = json.loads(base64.urlsafe_b64decode(segment + '=' * (-len(segment) % 4)))
        self.assertEqual(claims['scope'], ['GET ' + path])
        self.assertEqual(claims['exp'] - claims['iat'], 300)
        opener = Mock()
        opener.open.return_value = Response(json.dumps(document()).encode())
        with patch.object(asc.urllib.request, 'build_opener', return_value=opener) as factory:
            self.assertEqual(asc.read_build(jwt, '123', '1.0.1', '41'), ('VALID', False))
        request = opener.open.call_args.args[0]
        self.assertEqual(request.get_method(), 'GET')
        self.assertEqual(request.full_url, asc.ORIGIN + path)
        self.assertEqual(request.get_header('Authorization'), 'Bearer ' + jwt)
        self.assertIsNone(request.data)
        self.assertIsInstance(factory.call_args.args[1], asc.NoRedirect)
        handler = factory.call_args.args[2]
        self.assertTrue(handler._context.check_hostname)
        self.assertEqual(handler._context.verify_mode, asc.ssl.CERT_REQUIRED)
        opener.open.assert_called_once()

    def test_transport_limits_and_safe_no_retry(self):
        """Bound responses and sanitize HTTP/connection failures without retry."""
        path = asc.build_request_path('123', '1.0.1', '41')
        opener = Mock()
        opener.open.return_value = Response(b'x' * 65537)
        with patch.object(asc.urllib.request, 'build_opener', return_value=opener), self.assertRaises(asc.Refused):
            asc.read_response('SECRET_TOKEN', path)
        opener.open.assert_called_once()
        for error in [urllib.error.HTTPError('private-url', 401, 'SECRET_REASON', {}, io.BytesIO(b'PRIVATE_BODY')),
                      urllib.error.URLError('SECRET_REASON'), TimeoutError('SECRET_REASON')]:
            opener = Mock()
            opener.open.side_effect = error
            with patch.object(asc.urllib.request, 'build_opener', return_value=opener):
                with self.assertRaises(asc.Refused) as result:
                    asc.read_response('SECRET_TOKEN', path)
            self.assertNotIn('SECRET', str(result.exception))
            self.assertNotIn('private-url', str(result.exception))
            opener.open.assert_called_once()

    def test_run_consumes_secrets_and_prints_only_status(self):
        """Use two independently scoped tokens; never print identity or compliance claims."""
        env = {'ASC_API_KEY': '-----BEGIN PRIVATE KEY-----\nSYNTHETIC\n-----END PRIVATE KEY-----',
               'ASC_KEY_ID': 'ABCDEFGHIJ', 'ASC_TEAM_ID': 'KLMNOPQRST',
               'ASC_ISSUER_ID': '11111111-2222-3333-4444-555555555555'}
        with patch.dict(os.environ, env, clear=True), patch.object(asc, 'token', return_value='SECRET_TOKEN') as token, \
             patch.object(asc, 'read_app', return_value='123'), patch.object(asc, 'read_build', return_value=('VALID', False)), \
             contextlib.redirect_stdout(io.StringIO()) as output:
            self.assertEqual(asc.run(['--build-version', '1.0.1', '--build-number', '41']), 0)
            self.assertFalse(any(k.startswith('ASC_') for k in os.environ))
        self.assertEqual(token.call_count, 2)
        self.assertEqual(token.call_args.args[-1], asc.build_request_path('123', '1.0.1', '41'))
        self.assertIn('Apple processing state: VALID; expired: NO.', output.getvalue())
        self.assertIn('UNVERIFIED:', output.getvalue())
        self.assertNotIn('SECRET', output.getvalue())
        self.assertNotIn('123', output.getvalue())

    def test_bad_input_and_runtime_errors_safe(self):
        """Reject selectors before credential use; suppress arbitrary diagnostics."""
        with patch.object(asc, 'credentials') as credentials, contextlib.redirect_stdout(io.StringIO()) as output:
            self.assertEqual(asc.run(['SECRET_ARG']), 1)
        credentials.assert_not_called()
        self.assertNotIn('SECRET', output.getvalue())
        with patch.object(asc, 'credentials', side_effect=RuntimeError('SECRET_KEY')), \
             contextlib.redirect_stdout(io.StringIO()) as output:
            self.assertEqual(asc.run(['--build-version', '1.0.1', '--build-number', '41']), 1)
        self.assertNotIn('SECRET', output.getvalue())

    def assert_workflow(self, value):
        """Reject workflow mutations that move credentials outside guarded trusted-main code."""
        self.assertEqual(set(value['on']), {'workflow_dispatch'})
        self.assertEqual(value['permissions'], {'contents': 'read'})
        self.assertEqual(value['concurrency'], {'group': 'prosody-asc-preflight', 'cancel-in-progress': False})
        job = value['jobs']['read-existing-build']
        self.assertEqual(job['if'], "github.repository == 'narrate-it/prosody' && github.ref == 'refs/heads/main'")
        self.assertEqual(job['runs-on'], ['self-hosted', 'macOS', 'prosody'])
        self.assertEqual(job['environment'], 'prosody-asc')
        self.assertEqual(job['timeout-minutes'], 3)
        self.assertNotIn('env', job)
        self.assertNotIn('env', value)
        checkout, step = job['steps']
        self.assertEqual(checkout['uses'], 'actions/checkout@11d5960a326750d5838078e36cf38b85af677262')
        self.assertEqual(checkout['with'], {'ref': '${{ github.sha }}', 'persist-credentials': False, 'clean': False})
        self.assertNotIn('env', checkout)
        self.assertEqual(step['run'], '/usr/bin/python3 -I scripts/asc-preflight.py --build-version "$READ_VERSION" --build-number "$READ_BUILD"')
        expected = {k: '${{ secrets.' + k + ' }}' for k in ['ASC_API_KEY', 'ASC_ISSUER_ID', 'ASC_KEY_ID', 'ASC_TEAM_ID']}
        expected.update(READ_VERSION='${{ inputs.version }}', READ_BUILD='${{ inputs.build }}')
        self.assertEqual(step['env'], expected)
        inputs = value['on']['workflow_dispatch']['inputs']
        self.assertEqual(set(inputs), {'version', 'build'})
        for key, default in [('version', '1.0.1'), ('build', '41')]:
            self.assertEqual(inputs[key]['type'], 'string')
            self.assertTrue(inputs[key]['required'])
            self.assertEqual(inputs[key]['default'], default)

    def test_workflow_trust_and_secret_placement(self):
        """Use only the protected manual main-only self-hosted read job."""
        value = json.loads((Path(__file__).parents[1] / '.github/workflows/asc-build-status.yml').read_text())
        self.assert_workflow(value)
        mutations = [lambda d: d['jobs']['read-existing-build'].pop('if'),
                     lambda d: d.update(env={'ASC_API_KEY': '${{ secrets.ASC_API_KEY }}'}),
                     lambda d: d['jobs']['read-existing-build'].update(env={'ASC_API_KEY': 'SECRET'}),
                     lambda d: d['jobs']['read-existing-build'].update(environment='other'),
                     lambda d: d['jobs']['read-existing-build'].update({'runs-on': ['macos-latest']}),
                     lambda d: d['jobs']['read-existing-build']['steps'][0]['with'].update(ref='untrusted'),
                     lambda d: d['jobs']['read-existing-build']['steps'][1]['env'].update(PROSODY_KEYCHAIN_PASSWORD='SECRET'),
                     lambda d: d['on'].update(push={})]
        for mutate in mutations:
            bad = copy.deepcopy(value)
            mutate(bad)
            with self.assertRaises((AssertionError, KeyError)):
                self.assert_workflow(bad)


if __name__ == '__main__':
    unittest.main()
