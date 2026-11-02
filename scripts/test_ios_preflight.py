"""Hermetic CLI checks: no Flutter, signing, device or network operations."""
import json
import os
from pathlib import Path
import plistlib
import shutil
import shlex
import subprocess
import tempfile
import unittest


class IOSPreflightTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix='prosody-preflight-')
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        (self.root / 'scripts').mkdir()
        self.script = self.root / 'scripts/ios-preflight.sh'
        shutil.copyfile(Path(__file__).with_name('ios-preflight.sh'), self.script)
        self.bin = self.root / 'tools'
        self.bin.mkdir()
        # Tool discovery is allowed; executing these tools is not.
        for tool in ('flutter', 'xcrun', 'pod'):
            self.tool(tool, "echo unexpected-tool-execution >&2; exit 99")
        self.tool('python3', 'exec ' + shlex.quote(shutil.which('python3')) + ' "$@"')
        xcode = self.root / 'Xcode/usr/bin'
        xcode.mkdir(parents=True)
        (xcode / 'xcodebuild').touch()
        self.tool('xcode-select', "printf '%s\\n' '" + str(xcode.parent.parent) + "'")
        project = {'objects': {
            'target': {'isa': 'PBXNativeTarget', 'name': 'Runner', 'buildConfigurationList': 'configs'},
            'configs': {'buildConfigurations': ['release']},
            'release': {'name': 'Release', 'buildSettings': {'PRODUCT_BUNDLE_IDENTIFIER': 'run.prosody.prosody'}},
        }}
        self.tool('plutil', "cat <<'JSON'\n" + json.dumps(project) + '\nJSON')
        runner = self.root / 'mobile/ios/Runner'
        runner.mkdir(parents=True)
        with (runner / 'Info.plist').open('wb') as stream:
            plistlib.dump({'CFBundleIdentifier': '$(PRODUCT_BUNDLE_IDENTIFIER)',
                          'CFBundleShortVersionString': '$(FLUTTER_BUILD_NAME)',
                          'CFBundleVersion': '$(FLUTTER_BUILD_NUMBER)',
                          'UIBackgroundModes': ['audio']}, stream)
        (self.root / 'mobile/pubspec.yaml').write_text('version: 0.1.0+1\n')
        (self.root / 'mobile/lib').mkdir()
        (self.root / 'mobile/lib/main.dart').write_text('runApp(ProsodyApp(playback: playback, origin: origin));')

    def tool(self, name, body):
        path = self.bin / name
        path.write_text('#!/bin/sh\n' + body + '\n')
        path.chmod(0o755)

    def artifact(self, platform):
        path = self.root / f'mobile/build/ios/{platform}/Runner.app'
        path.mkdir(parents=True)
        (path / 'Runner').touch()
        (path / 'Info.plist').touch()

    def run_check(self, *args):
        result = subprocess.run(['/bin/sh', str(self.script), *args],
                                env={**os.environ, 'PATH': str(self.bin) + ':/usr/bin:/bin'},
                                text=True, capture_output=True, timeout=10)
        self.assertNotIn('unexpected-tool-execution', result.stderr)
        return result

    def test_simulator_cannot_satisfy_device_check(self):
        self.artifact('iphonesimulator')
        self.assertEqual(self.run_check('--preview').returncode, 0)
        device = self.run_check('--preview-device')
        self.assertEqual(device.returncode, 1)
        self.assertIn('Existing physical iPhone app missing', device.stdout)

    def test_device_artifact_passes_without_runtime_claim(self):
        self.artifact('iphoneos')
        result = self.run_check('--preview-device')
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn('runtime behavior is unverified', result.stdout)
        self.assertIn('does not establish Home Screen relaunch', result.stdout)
        self.assertEqual(self.run_check('--preview').returncode, 1)

    def test_artifacts_never_qualify_release(self):
        self.artifact('iphoneos')
        self.artifact('iphonesimulator')
        result = self.run_check()
        self.assertEqual(result.returncode, 1)
        self.assertIn('TestFlight requires separate recorded', result.stdout)

    def test_partial_device_artifact_and_unknown_option_fail(self):
        self.artifact('iphoneos')
        (self.root / 'mobile/build/ios/iphoneos/Runner.app/Runner').unlink()
        self.assertEqual(self.run_check('--preview-device').returncode, 1)
        self.assertEqual(self.run_check('--unknown').returncode, 2)


if __name__ == '__main__':
    unittest.main()
