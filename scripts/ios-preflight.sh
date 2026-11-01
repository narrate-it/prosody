#!/bin/sh
# Read-only source/artifact checks. Never builds, signs, uploads, reads .env,
# queries Keychain, or contacts Apple. A source check is not release acceptance.
set -eu
case "${1:-}" in
  '') mode=release ;;
  --preview) mode=preview ;;
  --help|-h)
    printf '%s\n' 'Usage: scripts/ios-preflight.sh [--preview]' \
      'Default: report TestFlight source gaps and unverified release gates (exit 1).' \
      '--preview: check prerequisites for the existing simulator binary; does not launch it.' \
      'Exit 0: preview prerequisites found; 1: blocked/unqualified; 2: tool/usage error.'
    exit 0 ;;
  *) printf '%s\n' 'Unknown option; use --help.' >&2; exit 2 ;;
esac
[ "$#" -le 1 ] || { printf '%s\n' 'Only one option is supported.' >&2; exit 2; }
command -v python3 >/dev/null 2>&1 || { printf '%s\n' 'python3 is required.' >&2; exit 2; }
repo_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
exec python3 - "$repo_dir" "$mode" <<'PY'
import json
import pathlib
import plistlib
import re
import shutil
import subprocess
import sys

root = pathlib.Path(sys.argv[1])
mode = sys.argv[2]
blocked = 0

def report(level, message):
    global blocked
    print(f'{level}: {message}')
    if level == 'BLOCK':
        blocked += 1

def text(relative):
    path = root / relative
    if not path.is_file():
        report('BLOCK', f'Missing project file: {relative}')
        return ''
    return path.read_text(encoding='utf-8')

print(f'Prosody iOS {mode} preflight — read-only')
for tool in ('flutter', 'xcrun', 'xcode-select', 'pod', 'plutil'):
    report('OK' if shutil.which(tool) else 'BLOCK', f'{tool} executable ' + ('found' if shutil.which(tool) else 'missing'))
if shutil.which('xcode-select'):
    selected = subprocess.run(['xcode-select', '-p'], capture_output=True, text=True, timeout=10)
    valid = selected.returncode == 0 and pathlib.Path(selected.stdout.strip(), 'usr/bin/xcodebuild').is_file()
    report('OK' if valid else 'BLOCK', 'Full Xcode developer directory ' + ('selected' if valid else 'not verified'))

pubspec = text('mobile/pubspec.yaml')
version = re.search(r'^version:\s*([0-9]+\.[0-9]+\.[0-9]+)\+([1-9][0-9]*)\s*$', pubspec, re.M)
if version:
    report('OK', f'Source version {version[1]}, build {version[2]}')
else:
    report('BLOCK', 'pubspec must contain a numeric version and positive build number')

info = {}
try:
    with (root / 'mobile/ios/Runner/Info.plist').open('rb') as f:
        info = plistlib.load(f)
except (OSError, ValueError, plistlib.InvalidFileException):
    report('BLOCK', 'Runner Info.plist is absent or invalid')
for key, expected in (
    ('CFBundleIdentifier', '$(PRODUCT_BUNDLE_IDENTIFIER)'),
    ('CFBundleShortVersionString', '$(FLUTTER_BUILD_NAME)'),
    ('CFBundleVersion', '$(FLUTTER_BUILD_NUMBER)'),
):
    report('OK' if info.get(key) == expected else 'BLOCK', f'{key} ' + ('uses expected build setting' if info.get(key) == expected else 'needs source review'))
report('OK' if 'audio' in info.get('UIBackgroundModes', []) else 'BLOCK', 'Background audio declaration ' + ('present' if 'audio' in info.get('UIBackgroundModes', []) else 'missing'))

release = {}
if shutil.which('plutil'):
    try:
        result = subprocess.run(['plutil', '-convert', 'json', '-o', '-', str(root / 'mobile/ios/Runner.xcodeproj/project.pbxproj')], capture_output=True, text=True, timeout=10)
        if result.returncode != 0:
            raise ValueError('invalid project')
        objects = json.loads(result.stdout)['objects']
        targets = [o for o in objects.values() if o.get('isa') == 'PBXNativeTarget' and o.get('name') == 'Runner']
        if len(targets) != 1:
            raise ValueError('missing or ambiguous target')
        ids = objects[targets[0]['buildConfigurationList']]['buildConfigurations']
        settings = [objects[i]['buildSettings'] for i in ids if objects[i].get('name') == 'Release']
        if len(settings) != 1:
            raise ValueError('missing or ambiguous release settings')
        release = settings[0]
    except (OSError, ValueError, KeyError, subprocess.TimeoutExpired):
        report('BLOCK', 'Runner Release configuration could not be inspected')
bundle = release.get('PRODUCT_BUNDLE_IDENTIFIER', '')
if isinstance(bundle, str) and re.fullmatch(r'[A-Za-z0-9-]+(?:\.[A-Za-z0-9-]+)+', bundle):
    report('OK', f'Runner Release bundle identifier: {bundle}')
else:
    report('BLOCK', 'Runner Release needs a concrete bundle identifier')
team = release.get('DEVELOPMENT_TEAM')
if team:
    report('NOTE', 'Runner Release contains a team setting; credential/profile validity was not inspected')
else:
    report('BLOCK' if mode == 'release' else 'NOTE', 'No team setting in Runner Release; external overrides/account state not inspected')

binary = root / 'mobile/build/ios/iphonesimulator/Runner.app'
if mode == 'preview':
    report('OK' if (binary / 'Runner').is_file() and (binary / 'Info.plist').is_file() else 'BLOCK', 'Existing simulator app ' + ('found' if (binary / 'Runner').is_file() and (binary / 'Info.plist').is_file() else 'missing'))
    report('NOTE', 'Artifact freshness, simulator boot state and live hot-reload attachment are not established by this check')
else:
    main = text('mobile/lib/main.dart')
    if 'runApp(ProsodyApp(playback: playback, origin: origin));' in main:
        report('BLOCK', 'Production composition supplies no human session adapter; sign-in is unavailable')
    else:
        report('UNVERIFIED', 'Human session composition and actual iOS authentication need acceptance')
    report('UNVERIFIED', 'AMOS current authority, worker execution and publication; end-to-end text-to-listening acceptance')
    report('UNVERIFIED', 'Signing identity/profile, Apple membership/roles, registered app and App Store Connect build-number history')
    report('UNVERIFIED', 'Release archive/device acceptance, privacy/export declarations, beta metadata and tester access')
    # Never convert static source checks into an automatic release approval.
    report('BLOCK', 'TestFlight requires separate recorded product, signing, account and release acceptance')

print('No build, signing, upload, network, Keychain or .env access performed.')
if blocked:
    print('Result: BLOCKED / NOT QUALIFIED. See docs/implementation/TESTFLIGHT.md.')
    sys.exit(1)
print('Result: local preview prerequisites found; runtime behavior is unverified.')
PY
