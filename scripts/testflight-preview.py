#!/usr/bin/env python3
"""One trusted self-hosted preview archive/upload; preserve private local evidence."""
import argparse
import base64
import contextlib
import ctypes
import datetime
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import plistlib
import re
import selectors
import stat
import signal
import ssl
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request

ROOT = Path(__file__).resolve().parent.parent
BUNDLE = 'run.sire.prosody'
ORIGIN = 'https://api.appstoreconnect.apple.com'


class Blocked(Exception):
    """Fixed safe release failure; raw tool/API diagnostics stay private."""


def load_asc():
    """Reuse reviewed credential/signer primitives from the trusted checkout."""
    spec = importlib.util.spec_from_file_location('asc', ROOT / 'scripts/asc-preflight.py')
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def request_token(asc, kid, issuer, key, path):
    """Sign one exact GET scope; no unscoped token or caller-provided endpoint."""
    now = int(time.time())
    encode = lambda x: asc.b64(json.dumps(x, separators=(',', ':')).encode())
    message = encode({'alg': 'ES256', 'kid': kid, 'typ': 'JWT'}) + b'.' + encode(
        {'iss': issuer, 'iat': now, 'exp': now + 300, 'aud': 'appstoreconnect-v1', 'scope': ['GET ' + path]})
    with tempfile.TemporaryDirectory(prefix='prosody-jwt-') as folder:
        p = Path(folder) / 'key.p8'
        p.write_bytes(key)
        p.chmod(0o600)
        signature = asc.raw_signature(asc.openssl(['dgst', '-sha256', '-sign', str(p)], message))
    return (message + b'.' + asc.b64(signature)).decode()


def api(asc, kid, issuer, key, path):
    """Read bounded Apple JSON once; never disclose remote bodies or retry."""
    if not path.startswith('/v1/') or len(path) > 2048 or '\n' in path or '\r' in path:
        raise Blocked('Invalid release metadata request')
    request = urllib.request.Request(ORIGIN + path, headers={
        'Authorization': 'Bearer ' + request_token(asc, kid, issuer, key, path), 'Accept': 'application/json'})
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), asc.NoRedirect(),
                                        urllib.request.HTTPSHandler(context=ssl.create_default_context()))
    try:
        with opener.open(request, timeout=20) as response:
            raw = response.read(1048577)
            if response.status != 200 or len(raw) > 1048576:
                raise Blocked('Apple metadata response refused')
        value = json.loads(raw)
        if not isinstance(value, dict) or not isinstance(value.get('links', {}), dict) or value.get('links', {}).get('next'):
            raise Blocked('Incomplete Apple metadata; no build number selected')
        return value
    except Blocked:
        raise
    except urllib.error.HTTPError as error:
        error.close()
        raise Blocked('Apple metadata request refused; no retry') from None
    except Exception:
        raise Blocked('Apple metadata request failed; no retry') from None


def version(value):
    """Accept bounded numeric marketing/build components for ordering."""
    if not isinstance(value, str) or not re.fullmatch(r'[0-9]{1,4}(?:\.[0-9]{1,2}){0,2}', value):
        raise Blocked('Unsupported existing version; inspect before release')
    return tuple([int(x) for x in value.split('.')] + [0] * (3 - len(value.split('.'))))


def select_numbers(source, versions, builds):
    """Choose an unused higher train/build from a complete bounded inventory."""
    current = version(source)
    previous = [version(x) for x in versions]
    if previous and max(previous) >= current:
        a, b, c = max(previous)
        if c >= 99:
            raise Blocked('Version rollover requires owner selection')
        current = (a, b, c + 1)
    n = max([0] + [version(x)[0] for x in builds]) + 1
    if n > 9999:
        raise Blocked('Build number rollover requires owner selection')
    return '.'.join(map(str, current)), str(n)


def capacity():
    """Admit one Mac phase using two interval idle/pressure/swap observations."""
    def snapshot():
        """Capture actual VM counters and normal-pressure indicator."""
        raw = subprocess.check_output(['vm_stat'], text=True, timeout=10)
        return (time.monotonic(), int(re.search(r'page size of (\d+)', raw)[1]),
                sum(map(int, re.findall(r'^(?:Swapins|Swapouts):\s+(\d+)', raw, re.M))),
                subprocess.check_output(['sysctl', '-n', 'kern.memorystatus_vm_pressure_level'], text=True, timeout=10).strip())
    process = subprocess.Popen(['top', '-l', '3', '-s', '5', '-n', '0'], stdout=subprocess.PIPE, text=True)
    try:
        states = [snapshot()]
        for _ in range(2):
            time.sleep(5)
            states.append(snapshot())
        raw = process.communicate(timeout=20)[0]
    finally:
        if process.poll() is None:
            process.terminate()
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait(timeout=5)
        process.stdout.close()
    idle = list(map(float, re.findall(r'([0-9.]+)% idle', raw)))
    loads = re.findall(r'Load Avg: ([^\n]+)', raw)
    rates = [(b[2] - a[2]) * b[1] / (b[0] - a[0]) for a, b in zip(states, states[1:])]
    thermal = subprocess.check_output(['pmset', '-g', 'therm'], text=True, timeout=10)
    if (len(idle) != 3 or len(loads) != 3 or sum(idle[-2:]) / 2 < 20 or any(s[3] != '1' for s in states)
            or all(x > 1048576 for x in rates) or 'No thermal warning level has been recorded' not in thermal
            or 'No performance warning level has been recorded' not in thermal):
        raise Blocked('Capacity HOLD; no automatic retry')
    return {'load': loads[-1], 'intervalIdle': idle[-2:],
            'swapBytesPerSecond': rates, 'pressure': 'normal'}


class Tools:
    """Finite owned process groups with private logs, never raw CI diagnostics."""
    def __init__(self, folder):
        """Bind all evidence to one private release directory."""
        self.folder = folder
        self.cleanup_unconfirmed = False
        self.unconfirmed_groups = []
        self.deadline = time.monotonic() + 3000
        self.env = dict(os.environ)
        self.env['COCOAPODS_DISABLE_STATS'] = 'true'

    def call(self, phase, argv, cwd=ROOT, timeout=120, log_limit=16777216):
        """Bound time/output and stop the entire owned group on every exit path."""
        remaining = min(timeout, self.deadline - time.monotonic())
        if remaining <= 0:
            raise Blocked('Release deadline reached; reconcile before retry')
        print('PHASE: ' + phase, flush=True)
        path = self.folder / (phase + '.log')
        end = time.monotonic() + remaining
        with path.open('wb') as log:
            child = subprocess.Popen(argv, cwd=cwd, env=self.env, stdout=subprocess.PIPE,
                                 stderr=subprocess.STDOUT, start_new_session=True)
            try:
                with selectors.DefaultSelector() as selector:
                    os.set_blocking(child.stdout.fileno(), False)
                    selector.register(child.stdout, selectors.EVENT_READ)
                    total = 0
                    while True:
                        if time.monotonic() >= end:
                            raise subprocess.TimeoutExpired(argv, remaining)
                        events = selector.select(min(0.2, end - time.monotonic()))
                        for _, _ in events:
                            data = os.read(child.stdout.fileno(), 65536)
                            if not data:
                                selector.unregister(child.stdout)
                                continue
                            total += len(data)
                            if total > log_limit:
                                raise Blocked(phase + ' output limit reached; no retry')
                            log.write(data)
                        if child.poll() is not None and (not events or not selector.get_map()):
                            break
                    code = child.wait(timeout=5)
            finally:
                try:
                    stop_group(child)
                except BaseException:
                    self.cleanup_unconfirmed = True
                    self.unconfirmed_groups.append(child.pid)
                    raise
                finally:
                    child.stdout.close()
        if code:
            raise Blocked(phase + ' failed; private log retained; no retry')
        return path


def stop_group(child):
    """Stop the exact owned group and verify no live members remain."""
    def live():
        """Use actual group/UID/state inventory; zombies cannot execute work."""
        child.poll()
        raw = subprocess.check_output(['/bin/ps', '-axo', 'pgid=,uid=,stat='], text=True, timeout=5)
        states = []
        for line in raw.splitlines():
            parts = line.split()
            if len(parts) == 3 and parts[0] == str(child.pid):
                if parts[1] != str(os.getuid()):
                    raise Blocked('Process-group custody changed; preserve lease')
                states.append(parts[2])
        return any(not state.startswith('Z') for state in states)
    if live():
        with contextlib.suppress(ProcessLookupError):
            os.killpg(child.pid, signal.SIGTERM)
    end = time.monotonic() + 3
    while live() and time.monotonic() < end:
        time.sleep(0.05)
    if live():
        with contextlib.suppress(ProcessLookupError):
            os.killpg(child.pid, signal.SIGKILL)
    child.wait(timeout=10)
    end = time.monotonic() + 3
    while live() and time.monotonic() < end:
        time.sleep(0.05)
    if live():
        raise Blocked('Owned process cleanup unconfirmed; preserve lease')


def artifact_digest(path):
    """Hash a bounded regular IPA through one non-symlink descriptor."""
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    with os.fdopen(fd, 'rb') as handle:
        before = os.fstat(handle.fileno())
        if not stat.S_ISREG(before.st_mode) or not 0 < before.st_size <= 4294967296:
            raise Blocked('IPA size/type refused')
        digest = hashlib.sha256()
        size = 0
        while True:
            block = handle.read(1048576)
            if not block:
                break
            size += len(block)
            if size > before.st_size:
                raise Blocked('IPA changed while hashing')
            digest.update(block)
        after = os.fstat(handle.fileno())
        if size != before.st_size or (before.st_size, before.st_mtime_ns, before.st_ctime_ns) != (after.st_size, after.st_mtime_ns, after.st_ctime_ns):
            raise Blocked('IPA changed while hashing')
    return digest.hexdigest()


def persist_receipt(folder, receipt):
    """Atomically fsync private evidence; a failed write never truncates prior bytes."""
    fd, temporary = tempfile.mkstemp(prefix='.receipt-', dir=folder)
    try:
        with os.fdopen(fd, 'w') as handle:
            json.dump(receipt, handle, indent=2)
            handle.flush()
            os.fsync(handle.fileno())
        os.replace(temporary, folder / 'receipt.json')
        directory = os.open(folder, os.O_RDONLY | os.O_DIRECTORY)
        try:
            os.fsync(directory)
        finally:
            os.close(directory)
    finally:
        with contextlib.suppress(FileNotFoundError):
            os.unlink(temporary)


def upload_once(tools, folder, receipt, ipa, auth):
    """Durably record uncertainty before the one external upload invocation."""
    receipt['uploadState'] = 'UNKNOWN: upload invocation entered; reconcile before retry'
    persist_receipt(folder, receipt)
    tools.call('apple-upload', ['xcrun', 'altool', '--upload-app', '-f', str(ipa), '-t', 'ios', *auth], timeout=600)
    receipt['uploadState'] = 'ACCEPTED: processing and tester availability not yet verified'
    persist_receipt(folder, receipt)


def require_unlocked_keychain(path):
    """Read lock metadata only; never unlock, authenticate or alter key access."""
    try:
        security = ctypes.CDLL('/System/Library/Frameworks/Security.framework/Security')
        core = ctypes.CDLL('/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation')
        security.SecKeychainOpen.argtypes = [ctypes.c_char_p, ctypes.POINTER(ctypes.c_void_p)]
        security.SecKeychainOpen.restype = ctypes.c_int32
        security.SecKeychainGetStatus.argtypes = [ctypes.c_void_p, ctypes.POINTER(ctypes.c_uint32)]
        security.SecKeychainGetStatus.restype = ctypes.c_int32
        core.CFRelease.argtypes = [ctypes.c_void_p]
        core.CFRelease.restype = None
        reference = ctypes.c_void_p()
        if security.SecKeychainOpen(os.fsencode(path), ctypes.byref(reference)) != 0 or not reference.value:
            raise Blocked('Signing keychain metadata unavailable; no signing attempted')
        try:
            status = ctypes.c_uint32()
            if security.SecKeychainGetStatus(reference, ctypes.byref(status)) != 0:
                raise Blocked('Signing keychain status unavailable; no signing attempted')
            if not status.value & 1:
                raise Blocked('prosody-ci is locked; owner must unlock before a new release phase')
        finally:
            core.CFRelease(reference)
    except Blocked:
        raise
    except Exception:
        raise Blocked('Signing keychain status unavailable; no signing attempted') from None


def signing_profile(team, keychain):
    """Select exactly one valid distribution identity/profile in owned custody."""
    raw = subprocess.check_output(['security', 'find-identity', '-v', '-p', 'codesigning', str(keychain)], text=True, timeout=10)
    identities = re.findall(r'\) ([0-9A-F]{40}) "Apple Distribution: [^"\n]+ \(' + re.escape(team) + r'\)"', raw)
    if len(identities) != 1:
        raise Blocked('Expected one new distribution identity; no authentication retry')
    matches = []
    for p in (Path.home() / 'Library/Developer/Xcode/UserData/Provisioning Profiles').glob('*.mobileprovision'):
        r = subprocess.run(['security', 'cms', '-D', '-i', str(p)], capture_output=True, timeout=10)
        if r.returncode:
            continue
        profile = plistlib.loads(r.stdout)
        ent = profile.get('Entitlements', {})
        certs = [hashlib.sha1(x).hexdigest().upper() for x in profile.get('DeveloperCertificates', [])]
        if (ent.get('application-identifier') == team + '.' + BUNDLE and identities[0] in certs
                and profile.get('TeamIdentifier') == [team] and ent.get('get-task-allow') is False
                and not profile.get('ProvisionedDevices') and not profile.get('ProvisionsAllDevices')
                and profile['ExpirationDate'] > datetime.datetime.now(datetime.timezone.utc).replace(tzinfo=None)):
            matches.append(profile)
    if len(matches) != 1 or not re.fullmatch(r'[0-9a-fA-F-]{36}', matches[0]['UUID']):
        raise Blocked('Expected one current matching App Store profile')
    return identities[0], matches[0]['UUID']


def archive_command(team, identity, profile, keychain, archive):
    """Keep app provisioning target-scoped; dependencies receive no profile."""
    return ['xcodebuild', '-workspace', 'Runner.xcworkspace', '-scheme', 'Runner',
            '-configuration', 'Release', '-destination', 'generic/platform=iOS', '-archivePath', str(archive),
            'DEVELOPMENT_TEAM=' + team, 'CODE_SIGN_STYLE=Manual', 'CODE_SIGN_IDENTITY=' + identity,
            'PROSODY_APP_PROFILE=' + profile, 'OTHER_CODE_SIGN_FLAGS=--keychain ' + str(keychain), 'archive']


def release(upload):
    """Build current preview on trusted main; upload only with explicit invocation."""
    os.umask(0o077)
    if (os.environ.get('GITHUB_REPOSITORY') != 'narrate-it/prosody'
            or os.environ.get('GITHUB_REF') != 'refs/heads/main' or os.environ.get('RUNNER_OS') != 'macOS'):
        raise Blocked('Trusted main self-hosted Mac context required')
    revision = os.environ.get('GITHUB_SHA', '')
    if not re.fullmatch('[0-9a-f]{40}', revision) or subprocess.check_output(['git', 'rev-parse', 'HEAD'], text=True).strip() != revision:
        raise Blocked('Checkout revision mismatch')
    if subprocess.check_output(['git', 'status', '--porcelain', '--untracked-files=no'], text=True):
        raise Blocked('Tracked source is modified')
    keychain = Path.home() / 'Library/Keychains/prosody-ci.keychain-db'
    require_unlocked_keychain(keychain)
    asc = load_asc()
    team = os.environ.get('ASC_TEAM_ID', '')
    kid, issuer, key = asc.credentials(os.environ)
    # Validate the key before any Apple request; this temporary signer file is removed.
    asc.token(kid, issuer, key)
    app = api(asc, kid, issuer, key, asc.APP_REQUEST_PATH)['data']
    if len(app) != 1 or app[0]['attributes']['bundleId'] != BUNDLE or not re.fullmatch('[0-9]+', app[0]['id']):
        raise Blocked('Existing app identity mismatch')
    appid = app[0]['id']
    builds = api(asc, kid, issuer, key, '/v1/builds?' + urllib.parse.urlencode(
        {'filter[app]': appid, 'include': 'preReleaseVersion', 'limit': '200'}))
    trains = api(asc, kid, issuer, key, '/v1/apps/' + appid + '/appStoreVersions?' + urllib.parse.urlencode(
        {'filter[platform]': 'IOS', 'limit': '200'}))
    source = re.search(r'^version:\s*([0-9]+\.[0-9]+\.[0-9]+)\+[0-9]+\s*$',
                       (ROOT / 'mobile/pubspec.yaml').read_text(), re.M)
    if not source:
        raise Blocked('Source version unavailable')
    name, number = select_numbers(source[1],
        [x['attributes']['versionString'] for x in trains['data']] +
        [x['attributes']['version'] for x in builds.get('included', []) if x['type'] == 'preReleaseVersions'],
        [x['attributes']['version'] for x in builds['data']])
    identity, profile = signing_profile(team, keychain)
    admitted = capacity()
    folder = Path(tempfile.mkdtemp(prefix='prosody-testflight-', dir=Path.home() / 'Library/Caches'))
    parent = os.open(folder.parent, os.O_RDONLY | os.O_DIRECTORY)
    try:
        os.fsync(parent)
    finally:
        os.close(parent)
    claim = Path.home() / '.agents/skills/claim/scripts/claim.sh'
    claim_env = dict(os.environ, CLAIM_REMOTE='/Users/Shared/mini-build-lease.git', CLAIM_PURPOSE='Prosody TestFlight preview archive/upload')
    result = subprocess.run([str(claim), 'claim', 'R-build-lease'], env=claim_env, capture_output=True, text=True, timeout=30)
    match = re.fullmatch(r'WON: R-build-lease ([0-9a-f]{40})\s*', result.stdout)
    if result.returncode or not match:
        raise Blocked('Build lease not acquired; no retry or foreign-token action')
    token = match[1]
    receipt = {'revision': revision, 'version': name, 'build': number, 'capacity': admitted,
               'uploadState': 'NOT ATTEMPTED', 'profile': profile, 'identity': identity, 'ownedLease': token}
    tools = None
    try:
        tools = Tools(folder)
        sdk = tools.call('sdk-version', ['xcodebuild', '-sdk', 'iphoneos', '-version', 'SDKVersion'])
        if sdk.stat().st_size > 4096 or not re.fullmatch(r'26(?:\.[0-9]+)*\s*', sdk.read_text()):
            raise Blocked('Expected installed iOS26 SDK')
        flutter = Path.home() / 'develop/flutter/bin/flutter'
        tools.call('locked-dependencies', [str(flutter), 'pub', 'get', '--enforce-lockfile'], ROOT / 'mobile', 180)
        tools.call('flutter-release', [str(flutter), 'build', 'ios', '--release', '--no-codesign',
                   '--build-name', name, '--build-number', number], ROOT / 'mobile', 1200)
        archive = folder / 'AgentProsody.xcarchive'
        require_unlocked_keychain(keychain)
        tools.call('signed-archive', archive_command(team, identity, profile, keychain, archive), ROOT / 'mobile/ios', 1200)
        tools.call('archive-metadata', ['/usr/bin/python3', '-I', str(ROOT / 'scripts/check-ios-archive.py'),
                   '--archive', str(archive), '--version', name, '--build', number])
        tools.call('archive-signature', ['codesign', '--verify', '--deep', '--strict', str(archive / 'Products/Applications/Runner.app')])
        if subprocess.check_output(['git', 'diff', '--name-only'], text=True):
            raise Blocked('Build changed tracked source; inspect before upload')
        options = folder / 'ExportOptions.plist'
        options.write_bytes(plistlib.dumps({'method': 'app-store-connect', 'destination': 'export',
            'teamID': team, 'signingStyle': 'manual', 'signingCertificate': identity,
            'provisioningProfiles': {BUNDLE: profile}, 'manageAppVersionAndBuildNumber': False,
            'uploadSymbols': True}))
        out = folder / 'export'
        require_unlocked_keychain(keychain)
        tools.call('export-ipa', ['xcodebuild', '-exportArchive', '-archivePath', str(archive),
                   '-exportPath', str(out), '-exportOptionsPlist', str(options)], timeout=300)
        ipas = list(out.glob('*.ipa'))
        if len(ipas) != 1:
            raise Blocked('Expected one exported IPA')
        receipt['ipaSHA256'] = artifact_digest(ipas[0])
        if upload:
            with tempfile.TemporaryDirectory(prefix='prosody-upload-key-') as secrets:
                p8 = Path(secrets) / ('AuthKey_' + kid + '.p8')
                p8.write_bytes(key)
                tools.env['API_PRIVATE_KEYS_DIR'] = secrets
                auth = ['--api-key', kid, '--api-issuer', issuer]
                tools.call('apple-validation', ['xcrun', 'altool', '--validate-app', '-f', str(ipas[0]), '-t', 'ios', *auth], timeout=300)
                upload_once(tools, folder, receipt, ipas[0], auth)
        print('PASS: Preview archive/export' + (' and upload accepted; processing remains unverified.' if upload else ' complete; no upload requested.'), flush=True)
    finally:
        try:
            if tools is not None and tools.cleanup_unconfirmed is True:
                receipt['unconfirmedProcessGroups'] = tools.unconfirmed_groups
            persist_receipt(folder, receipt)
        finally:
            if tools is not None and tools.cleanup_unconfirmed is True:
                raise Blocked('Owned process cleanup unconfirmed; lease retained in private receipt')
            released = subprocess.run([str(claim), 'release', 'R-build-lease', token], env=claim_env,
                                      capture_output=True, text=True, timeout=30)
            if released.returncode or 'RELEASED: R-build-lease' not in released.stdout:
                raise Blocked('Lease release unconfirmed; preserve exact owned token')


def main():
    """Run one explicit phase; never echo raw failures, secrets or private paths."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--upload', action='store_true')
    args = parser.parse_args()
    def stop(signum, frame):
        """Unwind owned child groups, credential files and lease on termination."""
        raise Blocked('Release interrupted; inspect retained receipt before retry')
    signal.signal(signal.SIGTERM, stop)
    try:
        release(args.upload)
    except Blocked as error:
        print('BLOCKED: ' + str(error), flush=True)
        return 1
    except Exception:
        print('BLOCKED: Release failed; private evidence retained; reconcile before retry', flush=True)
        return 1
    return 0


if __name__ == '__main__':
    sys.exit(main())
