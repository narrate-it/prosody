#!/usr/bin/env python3
"""Reconcile one existing approved runner; no installation, builds or signing."""
import argparse
import json
import os
from pathlib import Path
import plistlib
import signal
import socket
import subprocess
import sys
import time
from xml.parsers.expat import ExpatError

OLD = 'sirerun/prosody-legacy'
TARGET = 'narrate-it/prosody'
NAME = 'prosody-laptop-runner'
WORK = '_work-narrate-prosody'
OLD_URLS = {'https://github.com/sirerun/prosody', 'https://github.com/' + OLD}
TARGET_URL = 'https://github.com/' + TARGET
DEADLINE = time.monotonic() + 180


class Refused(Exception):
    """A safe diagnostic that contains no command output or credential values."""


def reap_owned(process, abnormal=False):
    """Stop and finitely reap only this operation's unfinished subprocess group."""
    if not abnormal and process.poll() is not None:
        return
    try:
        os.killpg(process.pid, signal.SIGTERM)
    except ProcessLookupError:
        pass
    try:
        process.communicate(timeout=1)
    except subprocess.TimeoutExpired:
        try:
            os.killpg(process.pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
        try:
            process.communicate(timeout=2)
        except subprocess.TimeoutExpired as error:
            raise Refused('owned subprocess cleanup unconfirmed; inspect before retry') from error


def command(args, cwd=None, env=None):
    """Run a finite subprocess without rendering its possibly sensitive output."""
    process = None
    completed = False
    try:
        remaining = DEADLINE - time.monotonic()
        if remaining <= 0:
            raise Refused('operation deadline reached')
        process = subprocess.Popen(args, cwd=cwd, env=env, stdout=subprocess.PIPE,
                                   stderr=subprocess.PIPE, text=True, start_new_session=True)
        try:
            output, _ = process.communicate(timeout=min(45, remaining))
            completed = process.returncode == 0
        except subprocess.TimeoutExpired as error:
            raise Refused('command deadline reached; inspect actual state before retry') from error
    except OSError as error:
        raise Refused('command unavailable or deadline reached; inspect actual state before retry') from error
    finally:
        if process is not None:
            reap_owned(process, abnormal=not completed)
    if process.returncode:
        raise Refused('command failed; inspect actual state before retry')
    return output


def github(path, method='GET'):
    """Read a GitHub response in memory; token responses are never printed."""
    try:
        return json.loads(command(['gh', 'api', '--method', method, path]))
    except Refused as error:
        raise Refused('GitHub ' + method + ' request failed for approved endpoint ' + path + '; inspect before retry') from error
    except ValueError as error:
        raise Refused('invalid GitHub response') from error


def mapping(value, message):
    """Reject malformed JSON/plist objects with a fixed safe refusal."""
    if not isinstance(value, dict):
        raise Refused(message)
    return value


def registration_token(path):
    """Require a nonempty token string without rendering the API response."""
    response = mapping(github(path, 'POST'), 'invalid GitHub token response')
    token = response.get('token')
    if (not isinstance(token, str) or not 1 <= len(token) <= 4096
            or any(not '!' <= char <= '~' for char in token)):
        raise Refused('invalid GitHub token response')
    return token


def repository_runners(repo):
    """Require a successful complete bounded inventory; errors never mean absence."""
    result = mapping(github('repos/' + repo + '/actions/runners?per_page=100'),
                     'runner inventory is not a JSON object')
    rows, count = result.get('runners'), result.get('total_count')
    if (not isinstance(rows, list) or any(not isinstance(row, dict) for row in rows)
            or type(count) is not int or not 0 <= count <= 100 or len(rows) != count):
        raise Refused('runner inventory is incomplete or exceeds bounded inspection')
    return rows


def binding(root):
    """Read noncredential local registration metadata without modifying it."""
    path = root / '.runner'
    if not path.is_file() or path.is_symlink():
        raise Refused('expected existing runner metadata is absent or linked')
    try:
        value = mapping(json.loads(path.read_text(encoding='utf-8-sig')),
                        'runner metadata is not a JSON object')
    except (OSError, ValueError) as error:
        raise Refused('runner metadata unreadable') from error
    if (value.get('agentName') != NAME or type(value.get('agentId')) is not int
            or value['agentId'] <= 0 or not isinstance(value.get('gitHubUrl'), str)):
        raise Refused('local runner identity does not match the approved runner')
    return value


def remote_match(rows, local):
    """Bind the GitHub runner ID and name, rejecting duplicates and missing IDs."""
    if not isinstance(rows, list) or any(not isinstance(row, dict) for row in rows):
        raise Refused('runner inventory contains a non-object row')
    matches = [r for r in rows if r.get('name') == NAME]
    if (len(matches) != 1 or type(matches[0].get('id')) is not int
            or matches[0]['id'] != local['agentId']):
        raise Refused('remote runner identity does not match local registration')
    return matches[0]


def target_match(rows, local):
    """Require the actual self-hosted macOS/X64/project labels on the target."""
    row = remote_match(rows, local)
    labels = row.get('labels')
    if (not isinstance(labels, list)
            or any(not isinstance(value, dict) or not isinstance(value.get('name'), str)
                   for value in labels)):
        raise Refused('target runner labels are malformed')
    labels = {value['name'] for value in labels}
    if not {'self-hosted', 'macOS', 'X64', 'prosody'}.issubset(labels):
        raise Refused('target runner labels do not match the admitted platform')
    return row


def no_local_runner():
    """Refuse any observed listener or worker instead of killing another job."""
    names = command(['ps', '-axo', 'comm=']).splitlines()
    if any(Path(name.strip()).name in {'Runner.Listener', 'Runner.Worker'} for name in names):
        raise Refused('local runner process observed; refuse reassignment')


def sync_directory(path):
    """Persist checkpoint/backup namespace changes before destructive work."""
    descriptor = os.open(path, os.O_RDONLY)
    try:
        os.fsync(descriptor)
    finally:
        os.close(descriptor)


def prepare_custody(path):
    """Persist newly created custody directories and their parent entries."""
    for directory in [Path.home() / '.local', Path.home() / '.local/state', path]:
        if directory.is_symlink():
            raise Refused('linked custody directory refused')
        if not directory.exists():
            directory.mkdir(mode=0o700)
            sync_directory(directory)
            sync_directory(directory.parent)
        if not directory.is_dir() or directory.stat().st_uid != os.getuid():
            raise Refused('custody directory ownership mismatch')
    os.chmod(path, 0o700)
    sync_directory(path)


def service(root):
    """Validate only the exact runner-owned LaunchAgent before any cleanup."""
    pointer = root / '.service'
    if not pointer.exists():
        return None
    if pointer.is_symlink():
        raise Refused('linked service metadata refused')
    path = Path(pointer.read_text().strip())
    if path.parent != Path.home() / 'Library/LaunchAgents' or path.is_symlink() or not path.is_file():
        raise Refused('service path is outside the owning LaunchAgents directory')
    if path.stat().st_uid != os.getuid():
        raise Refused('service has different ownership')
    try:
        data = mapping(plistlib.loads(path.read_bytes()), 'service metadata is not a plist dictionary')
    except (ValueError, ExpatError, plistlib.InvalidFileException) as error:
        raise Refused('service metadata is malformed') from error
    if data.get('WorkingDirectory') != str(root) or data.get('ProgramArguments') != [str(root / 'runsvc.sh')]:
        raise Refused('service does not point exclusively at this runner')
    label = data.get('Label', '')
    if not isinstance(label, str) or not label.startswith('actions.runner.') or not label.endswith('.' + NAME):
        raise Refused('service label does not match approved runner')
    return path, label


def loaded(label):
    """Inspect launchd membership without starting or stopping a service."""
    return label in {line.split()[-1] for line in command(['launchctl', 'list']).splitlines() if line.split()}


def register_command(root, args, token):
    """Pass one-use registration credentials through the supported runner environment."""
    if (not isinstance(token, str) or not 1 <= len(token) <= 4096
            or any(not '!' <= char <= '~' for char in token)):
        raise Refused('invalid registration token')
    env = os.environ.copy()
    env['ACTIONS_RUNNER_INPUT_TOKEN'] = token
    command([str(root / 'config.sh'), *args], cwd=root, env=env)


def reconcile(root, custody, apply=False):
    """Rebind only the approved idle runner, then qualify its target online status."""
    if sys.platform != 'darwin' or os.getuid() == 0 or root.is_symlink() or root != Path.home() / 'actions-runner-prosody':
        raise Refused('only the existing user-owned MacBook runner is admitted')
    if not os.environ.get('PROSODY_RUNNER_EXPECTED_HOST') or socket.gethostname() != os.environ['PROSODY_RUNNER_EXPECTED_HOST']:
        raise Refused('runner host differs from admitted host')
    for name in ['config.sh', 'svc.sh', 'bin/Runner.Listener']:
        path = root / name
        if not path.is_file() or path.is_symlink() or path.stat().st_uid != os.getuid():
            raise Refused('existing runner installation ownership mismatch')
    local = binding(root)
    current_url = local.get('gitHubUrl')
    if current_url not in OLD_URLS | {TARGET_URL}:
        raise Refused('runner belongs to a different repository')
    target_rows = repository_runners(TARGET)
    existing_service = service(root)
    stale_server = False
    if current_url in OLD_URLS:
        no_local_runner()
        old_rows = repository_runners(OLD)
        candidates = [r for r in old_rows if r.get('id') == local['agentId'] or r.get('name') == NAME]
        if candidates:
            old = remote_match(candidates, local)
            if old.get('busy') is not False or old.get('status') != 'offline':
                raise Refused('old runner must be explicitly idle and offline')
        else:
            stale_server = True
        if any(r.get('name') == NAME for r in target_rows):
            raise Refused('target name already exists; preserve its registration')
        if (root / WORK).exists() or (root / WORK).is_symlink():
            raise Refused('target work directory already exists; preserve its custody')
        action = 'reassign-existing-idle-runner'
    else:
        target_match(target_rows, local)
        if local.get('workFolder') != WORK:
            raise Refused('target work directory does not match the isolated profile')
        action = 'verify-existing-target-registration'
    if not apply:
        return {'action': action, 'target': TARGET, 'name': NAME, 'sourceServerAbsent': stale_server, 'effect': 'none'}
    if custody != Path.home() / '.local/state/prosody-runner-reassignment':
        raise Refused('only the dedicated runner custody directory is admitted')
    if any(path.is_symlink() for path in [Path.home() / '.local', Path.home() / '.local/state']):
        raise Refused('linked custody parents refused')
    prepare_custody(custody)
    record = custody / 'reassignment.json'
    original_id = local['agentId']
    if record.exists() or record.is_symlink():
        if record.is_symlink() or record.stat().st_uid != os.getuid():
            raise Refused('checkpoint ownership mismatch')
        try:
            previous = mapping(json.loads(record.read_text()), 'checkpoint is not a JSON object')
        except (OSError, ValueError) as error:
            raise Refused('checkpoint unreadable or malformed') from error
        if (previous.get('name') != NAME or previous.get('target') != TARGET
                or type(previous.get('originalID')) is not int or previous['originalID'] <= 0
                or type(previous.get('sourceServerAbsent', False)) is not bool):
            raise Refused('checkpoint belongs to a different handoff')
        original_id = previous['originalID']
        stale_server = previous.get('sourceServerAbsent', stale_server) if current_url == TARGET_URL else stale_server
        if current_url in OLD_URLS and original_id != local['agentId']:
            raise Refused('original registration differs from checkpoint')
    elif current_url == TARGET_URL:
        raise Refused('target registration lacks the owning handoff checkpoint')

    def checkpoint(phase):
        """Persist nonsecret recovery facts; failures require state reconciliation."""
        temporary = custody / ('checkpoint-' + str(os.getpid()) + '.tmp')
        fd = os.open(temporary, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(fd, 'w') as stream:
            json.dump({'phase': phase, 'name': NAME, 'target': TARGET, 'originalID': original_id, 'sourceServerAbsent': stale_server}, stream)
            stream.flush()
            os.fsync(stream.fileno())
        temporary.replace(record)
        sync_directory(custody)

    if current_url in OLD_URLS:
        if existing_service and loaded(existing_service[1]):
            raise Refused('legacy service is loaded; refuse live service mutation')
        # Prove both API permissions before changing the old local installation.
        remove = None if stale_server else registration_token('repos/' + OLD + '/actions/runners/remove-token')
        token = registration_token('repos/' + TARGET + '/actions/runners/registration-token')
        no_local_runner()
        checkpoint('remove-old-started')
        if existing_service:
            path, label = existing_service
            if loaded(label):
                raise Refused('legacy service is loaded; refuse live service mutation')
            # Both paths were validated above. Preserve the public service definition,
            # not credential files, before clearing this exact stopped registration.
            backup = custody / 'old-launchagent.plist'
            fd = os.open(backup, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
            with os.fdopen(fd, 'wb') as stream:
                stream.write(path.read_bytes())
                stream.flush()
                os.fsync(stream.fileno())
            sync_directory(custody)
            path.unlink()
            (root / '.service').unlink()
        if stale_server:
            # This branch is selected only from a complete successful source
            # inventory, never as a fallback after denied/uncertain server removal.
            command([str(root / 'config.sh'), 'remove', '--unattended', '--local'], cwd=root)
        else:
            register_command(root, ['remove', '--unattended'], remove)
        checkpoint('old-removed')
        checkpoint('register-target-started')
        register_command(root, ['--unattended', '--url', TARGET_URL, '--name', NAME, '--work', WORK, '--labels', 'prosody'], token)
        checkpoint('target-registered')
    actual = binding(root)
    if actual.get('gitHubUrl') != TARGET_URL or actual.get('workFolder') != WORK:
        raise Refused('new local repository binding not confirmed')
    target_match(repository_runners(TARGET), actual)
    if not service(root):
        command([str(root / 'svc.sh'), 'install'], cwd=root)
    current_service = service(root)
    if not current_service:
        raise Refused('target service installation not confirmed')
    if not loaded(current_service[1]):
        command([str(root / 'svc.sh'), 'start'], cwd=root)
    for _ in range(8):
        row = github('repos/' + TARGET + '/actions/runners/' + str(actual['agentId']))
        target_match([row], actual)
        if row.get('status') == 'online':
            checkpoint('online')
            return {'name': NAME, 'target': TARGET, 'id': actual['agentId'], 'status': 'online'}
        time.sleep(2)
    checkpoint('registered-online-unconfirmed')
    raise Refused('target registered; online state unconfirmed; do not repeat registration')


def main():
    """Select a read-only plan or the explicitly approved IaC apply operation."""
    parser = argparse.ArgumentParser(description=__doc__)
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument('--plan', action='store_true')
    group.add_argument('--apply', action='store_true')
    group.add_argument('--refuse-destroy', action='store_true')
    args = parser.parse_args()
    if args.refuse_destroy:
        raise Refused('runner removal requires a separate reviewed custody action')
    root = Path(os.environ['PROSODY_RUNNER_DIR'])
    custody = Path(os.environ['PROSODY_RUNNER_CUSTODY'])
    print(json.dumps(reconcile(root, custody, args.apply)))


if __name__ == '__main__':
    def interrupted(_signum, _frame):
        """Unwind through subprocess cleanup when an owning supervisor stops us."""
        raise Refused('operation interrupted; inspect actual state before retry')

    signal.signal(signal.SIGTERM, interrupted)
    try:
        main()
    except Refused as error:
        print('runner reassignment: ' + str(error), file=sys.stderr)
        sys.exit(1)
    except (OSError, ValueError, KeyError, KeyboardInterrupt):
        print('runner reassignment refused or incomplete; inspect actual registration/service and private checkpoint before retry', file=sys.stderr)
        sys.exit(1)
