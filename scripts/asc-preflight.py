#!/usr/bin/env python3
"""Read only the existing ASC app; never configure signing or upload a build."""
import base64
import json
import math
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
APP_REQUEST_PATH = '/v1/apps?' + urllib.parse.urlencode(
    {'filter[bundleId]': BUNDLE, 'limit': '2', 'fields[apps]': 'bundleId'})


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


def build_request_path(appid, version, build):
    """Construct one canonical, exact iOS build metadata GET selection."""
    if (type(appid) is not str or not re.fullmatch(r'[1-9][0-9]{0,19}', appid)
            or type(version) is not str
            or not re.fullmatch(r'(?:0|[1-9][0-9]{0,2})(?:\.(?:0|[1-9][0-9]{0,2})){2}', version)
            or type(build) is not str or not re.fullmatch(r'[1-9][0-9]{0,17}', build)):
        raise Refused('Build selection missing or malformed')
    return '/v1/builds?' + urllib.parse.urlencode({
        'filter[app]': appid, 'filter[preReleaseVersion.version]': version,
        'filter[preReleaseVersion.platform]': 'IOS', 'filter[version]': build,
        'include': 'preReleaseVersion',
        'fields[builds]': 'version,processingState,expired,preReleaseVersion,app',
        'fields[preReleaseVersions]': 'version,platform', 'limit': '2'})


def request_scope(path):
    """Allow only the original app query or a canonical exact-build query."""
    if path == APP_REQUEST_PATH:
        return path
    try:
        if type(path) is not str or len(path) > 2048:
            raise ValueError()
        target = urllib.parse.urlsplit(path)
        query = urllib.parse.parse_qs(target.query, keep_blank_values=True, strict_parsing=True)
        keys = ['filter[app]', 'filter[preReleaseVersion.version]', 'filter[version]']
        if any(len(query[k]) != 1 for k in keys):
            raise ValueError()
        expected = build_request_path(*(query[k][0] for k in keys))
        if path != expected:
            raise ValueError()
    except (ValueError, KeyError, TypeError):
        raise Refused('Invalid ASC request scope') from None
    return path


def token(kid, issuer, key, path=APP_REQUEST_PATH):
    """Create a five-minute exact GET-scoped token in an ephemeral directory."""
    path = request_scope(path)
    now = int(time.time())
    header = {'alg': 'ES256', 'kid': kid, 'typ': 'JWT'}
    claims = {'iss': issuer, 'iat': now, 'exp': now + 300,
              'aud': 'appstoreconnect-v1', 'scope': ['GET ' + path]}
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


def read_response(bearer, path):
    """Read at most 64KiB over verified TLS once; never follow remote links."""
    path = request_scope(path)
    request = urllib.request.Request(ORIGIN + path,
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
        return body
    except urllib.error.HTTPError as error:
        code = error.code
        error.close()
        if type(code) is int and 100 <= code <= 599:
            raise Refused('ASC HTTP status ' + str(code) + '; inspect account access before retry') from None
        raise Refused('ASC HTTP status unavailable; inspect before retry') from None
    except (urllib.error.URLError, TimeoutError, OSError):
        raise Refused('ASC connection failed; diagnostics suppressed') from None


def read_app(bearer):
    """Validate the existing bundle and return its ID privately to the caller."""
    try:
        value = json.loads(read_response(bearer, APP_REQUEST_PATH))
        rows = value['data']
        links = value.get('links', {})
        if not isinstance(links, dict):
            raise ValueError()
        if links.get('next') or not isinstance(rows, list) or len(rows) != 1:
            raise Refused('Expected exactly one existing app')
        app = rows[0]
        if app['type'] != 'apps' or not re.fullmatch(r'[0-9]+', app['id']) or app['attributes']['bundleId'] != BUNDLE:
            raise Refused('Existing app identity mismatch')
        return app['id']
    except (ValueError, KeyError, TypeError):
        raise Refused('ASC response invalid; diagnostics suppressed') from None


def unique_pairs(pairs):
    """Reject decoded duplicate JSON keys without rendering their contents."""
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError()
        result[key] = value
    return result


def reject_constant(value):
    """Refuse non-JSON NaN and infinity constants without printing them."""
    raise ValueError()


def finite_float(value):
    """Reject numeric overflow without rendering response contents."""
    number = float(value)
    if not math.isfinite(number):
        raise ValueError()
    return number


def read_build(bearer, appid, version, build):
    """Match app, iOS train and build; return only finite processing/expiry flags."""
    path = build_request_path(appid, version, build)
    raw = read_response(bearer, path)
    try:
        value = json.loads(raw.decode('utf-8'), object_pairs_hook=unique_pairs,
                           parse_constant=reject_constant, parse_float=finite_float)
        rows, included = value['data'], value['included']
        links = value.get('links', {})
        if (type(links) is not dict or links.get('next') or type(rows) is not list
                or len(rows) != 1 or type(included) is not list or len(included) != 1):
            raise ValueError()
        row, train = rows[0], included[0]
        resource_id = lambda v: type(v) is str and re.fullmatch(r'[A-Za-z0-9-]{1,128}', v)
        if row['type'] != 'builds' or not resource_id(row['id']):
            raise ValueError()
        attrs, relations = row['attributes'], row['relationships']
        if relations['app']['data'] != {'type': 'apps', 'id': appid}:
            raise ValueError()
        if (train['type'] != 'preReleaseVersions' or not resource_id(train['id'])
                or relations['preReleaseVersion']['data'] != {'type': train['type'], 'id': train['id']}
                or train['attributes']['version'] != version or train['attributes']['platform'] != 'IOS'
                or attrs['version'] != build or type(attrs['expired']) is not bool
                or type(attrs['processingState']) is not str
                or attrs['processingState'] not in ('PROCESSING', 'FAILED', 'INVALID', 'VALID')):
            raise ValueError()
        return attrs['processingState'], attrs['expired']
    except (ValueError, KeyError, TypeError, RecursionError):
        raise Refused('Exact iOS build metadata unavailable; reconcile without upload') from None


def selection(argv):
    """Accept only bounded explicit numeric selectors; never echo bad arguments."""
    if not argv:
        return None
    if (type(argv) not in (list, tuple) or len(argv) != 4
            or argv[0] != '--build-version' or argv[2] != '--build-number'):
        raise Refused('Build selection missing or malformed')
    build_request_path('1', argv[1], argv[3])
    return argv[1], argv[3]


def run(argv=None):
    """Print only fixed evidence; failures never include key/JWT/API bodies."""
    try:
        wanted = selection(argv)
        kid, issuer, key = credentials(os.environ)
        appid = read_app(token(kid, issuer, key))
        if wanted is not None:
            version, build = wanted
            path = build_request_path(appid, version, build)
            state, expired = read_build(token(kid, issuer, key, path), appid, version, build)
            print('OBSERVED: iOS version ' + version + ', build ' + build + '.')
            print('Apple processing state: ' + state + '; expired: ' + ('YES' if expired else 'NO') + '.')
            print('UNVERIFIED: Team association, export compliance, tester membership, visibility and device installation.')
            return 0
    except Refused as error:
        print('BLOCKED: ' + str(error))
        return 1
    except Exception:
        print('BLOCKED: Unexpected preflight failure; diagnostics suppressed')
        return 1
    print('PASS: Existing run.sire.prosody app is readable by configured ASC key.')
    print('UNVERIFIED: Team association, build-number history, signing, upload and TestFlight acceptance.')
    return 0


def main(argv=None):
    """Unwind normal termination through private temporary-file cleanup."""
    def terminate(signum, frame):
        raise Refused('Preflight terminated')
    previous = signal.signal(signal.SIGTERM, terminate)
    try:
        return run(argv)
    finally:
        signal.signal(signal.SIGTERM, previous)


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
