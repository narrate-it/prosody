#!/usr/bin/env python3
"""Read only the existing ASC app; never configure signing or upload a build."""
import base64
import json
import os
import re
import signal
import ssl
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path

BUNDLE = 'run.sire.prosody'
ORIGIN = 'https://api.appstoreconnect.apple.com'


class Refused(Exception):
    """A fixed, non-sensitive refusal safe for public CI output."""


def b64(data):
    """Encode a JWT segment without padding."""
    return base64.urlsafe_b64encode(data).rstrip(b'=')


def raw_signature(der):
    """Convert a strictly bounded P-256 ASN.1 signature to JOSE r||s."""
    if not 8 <= len(der) <= 72 or der[:1] != b'\x30' or der[1] != len(der) - 2:
        raise Refused('Invalid signer output')
    values = []
    offset = 2
    for _ in range(2):
        if offset + 2 > len(der) or der[offset] != 2:
            raise Refused('Invalid signer output')
        length = der[offset + 1]
        value = der[offset + 2:offset + 2 + length]
        if not 1 <= length <= 33 or len(value) != length or value[0] & 128:
            raise Refused('Invalid signer output')
        if len(value) > 1 and value[0] == 0:
            if not value[1] & 128:
                raise Refused('Invalid signer output')
            value = value[1:]
        if len(value) > 32 or not any(value):
            raise Refused('Invalid signer output')
        values.append(value.rjust(32, b'\0'))
        offset += length + 2
    if offset != len(der):
        raise Refused('Invalid signer output')
    return b''.join(values)


def credentials(env):
    """Consume the four configured secrets without rendering their values."""
    values = {key: env.pop(key, '') for key in
              ('ASC_API_KEY', 'ASC_ISSUER_ID', 'ASC_KEY_ID', 'ASC_TEAM_ID')}
    if not re.fullmatch(r'[A-Za-z0-9]{10}', values['ASC_KEY_ID']):
        raise Refused('ASC key ID missing or malformed')
    if not re.fullmatch(r'[A-Za-z0-9]{10}', values['ASC_TEAM_ID']):
        raise Refused('ASC team ID missing or malformed')
    if not re.fullmatch(r'[0-9a-fA-F]{8}(?:-[0-9a-fA-F]{4}){3}-[0-9a-fA-F]{12}', values['ASC_ISSUER_ID']):
        raise Refused('ASC issuer ID missing or malformed')
    secret = values['ASC_API_KEY']
    if not secret or len(secret) > 16384:
        raise Refused('ASC private key missing or malformed')
    try:
        key = secret.strip().encode('ascii')
        if not key.startswith(b'-----BEGIN PRIVATE KEY-----'):
            key = base64.b64decode(key, validate=True)
        if not key.startswith(b'-----BEGIN PRIVATE KEY-----\n') or not key.rstrip().endswith(b'-----END PRIVATE KEY-----'):
            raise ValueError()
    except (ValueError, UnicodeError):
        raise Refused('ASC private key missing or malformed') from None
    return values['ASC_KEY_ID'], values['ASC_ISSUER_ID'], key


def openssl(args, data=None):
    """Run the installed system signer with bounded, captured diagnostics."""
    try:
        result = subprocess.run(['/usr/bin/openssl', *args], input=data,
                                capture_output=True, timeout=10, check=False)
    except (OSError, subprocess.TimeoutExpired):
        raise Refused('System signer unavailable') from None
    if result.returncode:
        raise Refused('System signer rejected key')
    return result.stdout


def token(kid, issuer, key):
    """Create a five-minute, read-only app-list token in an ephemeral directory."""
    now = int(time.time())
    header = {'alg': 'ES256', 'kid': kid, 'typ': 'JWT'}
    claims = {'iss': issuer, 'iat': now, 'exp': now + 300,
              'aud': 'appstoreconnect-v1', 'scope': ['GET /v1/apps']}
    encode = lambda value: b64(json.dumps(value, separators=(',', ':')).encode())
    message = encode(header) + b'.' + encode(claims)
    with tempfile.TemporaryDirectory(prefix='prosody-asc-') as directory:
        path = Path(directory) / 'key.p8'
        fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(fd, 'wb') as handle:
            handle.write(key)
        # Validate canonical public SPKI bytes, not platform-specific private-key text.
        public = openssl(['pkey', '-in', str(path), '-pubout', '-outform', 'DER'])
        prefix = bytes.fromhex('3059301306072a8648ce3d020106082a8648ce3d03010703420004')
        if len(public) != 91 or not public.startswith(prefix):
            raise Refused('ASC key must use P-256')
        signature = raw_signature(openssl(['dgst', '-sha256', '-sign', str(path)], message))
    return (message + b'.' + b64(signature)).decode('ascii')


class NoRedirect(urllib.request.HTTPRedirectHandler):
    """Never forward the bearer credential to a redirected destination."""

    def redirect_request(self, req, fp, code, msg, headers, newurl):
        """Refuse every redirect, including same-origin redirects."""
        raise Refused('ASC redirect refused')


def read_app(bearer):
    """Issue one bounded TLS GET; discard all remote diagnostics and metadata."""
    query = urllib.parse.urlencode({'filter[bundleId]': BUNDLE, 'limit': '2',
                                  'fields[apps]': 'bundleId'})
    request = urllib.request.Request(ORIGIN + '/v1/apps?' + query,
                                    headers={'Authorization': 'Bearer ' + bearer,
                                             'Accept': 'application/json'}, method='GET')
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect(),
                                        urllib.request.HTTPSHandler(context=ssl.create_default_context()))
    try:
        with opener.open(request, timeout=15) as response:
            if response.status != 200:
                raise Refused('ASC did not return success')
            body = response.read(65537)
            if len(body) > 65536:
                raise Refused('ASC response exceeded bound')
        value = json.loads(body)
        rows = value['data']
        if value.get('links', {}).get('next') or not isinstance(rows, list) or len(rows) != 1:
            raise Refused('Expected exactly one existing app')
        app = rows[0]
        if app['type'] != 'apps' or not re.fullmatch(r'[0-9]+', app['id']) or app['attributes']['bundleId'] != BUNDLE:
            raise Refused('Existing app identity mismatch')
    except urllib.error.HTTPError as error:
        error.close()
        raise Refused('ASC HTTP access denied or unavailable') from None
    except (urllib.error.URLError, TimeoutError, OSError, ValueError, KeyError, TypeError):
        raise Refused('ASC response unavailable or invalid') from None


def run():
    """Print only fixed evidence; failures never include key/JWT/API bodies."""
    try:
        kid, issuer, key = credentials(os.environ)
        read_app(token(kid, issuer, key))
    except Refused as error:
        print('BLOCKED: ' + str(error))
        return 1
    except Exception:
        print('BLOCKED: Unexpected preflight failure; diagnostics suppressed')
        return 1
    print('PASS: Existing run.sire.prosody app is readable by configured ASC key.')
    print('UNVERIFIED: Team association, build-number history, signing, upload and TestFlight acceptance.')
    return 0


def main():
    """Unwind normal termination through private temporary-file cleanup."""
    def terminate(signum, frame):
        raise Refused('Preflight terminated')
    previous = signal.signal(signal.SIGTERM, terminate)
    try:
        return run()
    finally:
        signal.signal(signal.SIGTERM, previous)


if __name__ == '__main__':
    sys.exit(main())
