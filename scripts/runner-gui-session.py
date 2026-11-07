#!/usr/bin/env python3
"""Opt one existing approved idle runner into its logged-in GUI security session."""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import plistlib
import socket
import signal
import sys
import time

SPEC = importlib.util.spec_from_file_location('runner_owner', Path(__file__).with_name('reassign-prosody-runner.py'))
OWNER = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(OWNER)


def desired_bytes(raw):
    """Change only a validated Boolean SessionCreate field, preserving every other value."""
    if len(raw) > 131072:
        raise OWNER.Refused('service definition exceeds bound')
    try:
        value = plistlib.loads(raw)
    except Exception as error:
        raise OWNER.Refused('service definition is malformed') from error
    if not isinstance(value, dict) or type(value.get('SessionCreate')) is not bool:
        raise OWNER.Refused('explicit Boolean SessionCreate required')
    if value['SessionCreate'] is False:
        return raw
    value['SessionCreate'] = False
    return plistlib.dumps(value, sort_keys=False)


def atomic_write(path, raw):
    """Persist an owned exclusive temporary and atomically replace the destination."""
    temporary = path.with_name(path.name + '.prosody-' + str(os.getpid()))
    fd = os.open(temporary, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(fd, 'wb') as stream:
        stream.write(raw)
        stream.flush()
        os.fsync(stream.fileno())
    temporary.replace(path)
    OWNER.sync_directory(path.parent)


def no_worker(root):
    """Refuse any live job worker; no processes are killed by this control."""
    rows = OWNER.command(['ps', '-axo', 'comm=']).splitlines()
    if any(Path(row.strip()).name == 'Runner.Worker' for row in rows):
        raise OWNER.Refused('job worker observed; preserve running work')


def reconcile(root, custody, apply=False):
    """Validate exact custody, retain a recovery checkpoint and finitely restart only this agent."""
    expected = Path.home() / 'actions-runner-prosody'
    if (sys.platform != 'darwin' or os.getuid() == 0 or root != expected or root.is_symlink()
            or not os.environ.get('PROSODY_RUNNER_EXPECTED_HOST')
            or socket.gethostname() != os.environ['PROSODY_RUNNER_EXPECTED_HOST']):
        raise OWNER.Refused('approved user-owned MacBook host required')
    if custody != Path.home() / '.local/state/prosody-runner-gui-session':
        raise OWNER.Refused('exact dedicated custody directory required')
    local = OWNER.binding(root)
    if local.get('gitHubUrl') != OWNER.TARGET_URL or local.get('workFolder') != OWNER.WORK:
        raise OWNER.Refused('existing target registration required; no reassignment')
    row = OWNER.target_match(OWNER.repository_runners(OWNER.TARGET), local)
    if row.get('busy') is not False or row.get('status') != 'online':
        raise OWNER.Refused('existing runner must be explicitly online and idle')
    current = OWNER.service(root)
    if current is None:
        raise OWNER.Refused('existing runner service required')
    path, label = current
    raw = path.read_bytes()
    desired = desired_bytes(raw)
    domain = 'gui/' + str(os.getuid())
    OWNER.command(['launchctl', 'print', domain])
    OWNER.command(['launchctl', 'print', domain + '/' + label])
    no_worker(root)
    digest = hashlib.sha256(raw).hexdigest()
    result = {'action': 'verify-gui-session' if desired == raw else 'use-gui-session',
              'name': OWNER.NAME, 'runnerID': local['agentId'], 'originalSHA256': digest,
              'desiredSHA256': hashlib.sha256(desired).hexdigest(), 'effect': 'none'}
    record = custody / 'session-change.json'
    if record.exists() or record.is_symlink():
        if record.is_symlink() or record.stat().st_uid != os.getuid():
            raise OWNER.Refused('prior session checkpoint custody mismatch')
        try:
            prior = json.loads(record.read_text())
        except (ValueError, OSError) as error:
            raise OWNER.Refused('prior session checkpoint unreadable') from error
        if (not isinstance(prior, dict) or prior.get('name') != OWNER.NAME
                or prior.get('runnerID') != local['agentId']
                or prior.get('phase') != 'online; signing-unqualified'
                or prior.get('desiredSHA256') != digest or desired != raw):
            raise OWNER.Refused('prior session attempt needs explicit reconciliation; no retry')
    if not apply or desired == raw:
        return result
    if any(p.is_symlink() for p in [Path.home() / '.local', Path.home() / '.local/state', custody]):
        raise OWNER.Refused('linked custody refused')
    OWNER.prepare_custody(custody)
    backup = custody / ('launchagent-' + digest + '.plist')
    if backup.exists():
        if backup.is_symlink() or backup.stat().st_uid != os.getuid() or backup.read_bytes() != raw:
            raise OWNER.Refused('backup custody mismatch')
    else:
        fd = os.open(backup, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(fd, 'wb') as stream:
            stream.write(raw)
            stream.flush()
            os.fsync(stream.fileno())
        OWNER.sync_directory(custody)

    def checkpoint(phase):
        """Record partial effects without claiming rollback or recovery success."""
        atomic_write(custody / 'session-change.json', json.dumps({**result, 'phase': phase}).encode())

    checkpoint('prepared')
    # Re-read immediately before stop. No registration, token, ACL or keychain mutation.
    fresh = OWNER.target_match(OWNER.repository_runners(OWNER.TARGET), local)
    if fresh.get('busy') is not False or fresh.get('status') != 'online':
        raise OWNER.Refused('runner became busy or offline; no stop attempted')
    no_worker(root)
    if OWNER.service(root) != current or path.read_bytes() != raw:
        raise OWNER.Refused('service changed during admission')
    checkpoint('stop-started')
    OWNER.command(['launchctl', 'bootout', domain + '/' + label])
    checkpoint('stopped')
    OWNER.no_local_runner()
    if OWNER.service(root) != current or path.read_bytes() != raw:
        raise OWNER.Refused('stopped service drifted; retain checkpoint')
    atomic_write(path, desired)
    checkpoint('definition-written')
    OWNER.command(['launchctl', 'bootstrap', domain, str(path)])
    checkpoint('started')
    for _ in range(8):
        row = OWNER.target_match(OWNER.repository_runners(OWNER.TARGET), local)
        if row.get('status') == 'online':
            OWNER.command(['launchctl', 'print', domain + '/' + label])
            if path.read_bytes() != desired:
                raise OWNER.Refused('service definition drifted after start')
            checkpoint('online; signing-unqualified')
            return {**result, 'effect': 'runner-restarted', 'status': 'online', 'signing': 'NOT TESTED'}
        time.sleep(2)
    raise OWNER.Refused('restart online state unconfirmed; inspect checkpoint, no automatic retry')


def main():
    """Provide separate read-only plan and explicit apply entry points for protected IaC."""
    parser = argparse.ArgumentParser(description=__doc__)
    flags = parser.add_mutually_exclusive_group(required=True)
    flags.add_argument('--plan', action='store_true')
    flags.add_argument('--apply', action='store_true')
    flags.add_argument('--refuse-destroy', action='store_true')
    args = parser.parse_args()
    try:
        if args.refuse_destroy:
            raise OWNER.Refused('automatic service restoration/destroy refused')
        result = reconcile(Path(os.environ['PROSODY_RUNNER_DIR']),
                           Path(os.environ['PROSODY_RUNNER_GUI_CUSTODY']), args.apply)
        print(json.dumps(result))
    except (OWNER.Refused, OSError, KeyError, KeyboardInterrupt):
        print('GUI session operation refused/incomplete; inspect retained checkpoint before retry', file=sys.stderr)
        return 1
    return 0


def interrupted(_signum, _frame):
    """Unwind through the owned subprocess cleanup on supervisor termination."""
    raise OWNER.Refused('operation interrupted; inspect checkpoint before retry')


if __name__ == '__main__':
    signal.signal(signal.SIGTERM, interrupted)
    sys.exit(main())
