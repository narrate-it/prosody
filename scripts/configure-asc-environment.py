#!/usr/bin/env python3
"""IaC-only main-branch environment setup; no secret reads/writes or signing."""
import argparse
import json
import re
import subprocess
import sys
import time

REPOSITORY = 'narrate-it/prosody'
REPOSITORY_ID = 1440533867
ENVIRONMENT = 'prosody-asc'
ROOT = 'repos/' + REPOSITORY
ENV = ROOT + '/environments/' + ENVIRONMENT
POLICY = {'protected_branches': False, 'custom_branch_policies': True}
DEADLINE = time.monotonic() + 90


class Refused(Exception):
    """A fixed diagnostic that never includes GitHub bodies or credentials."""


def api(path, method='GET', data=None, allow_absent=False):
    """Make one finite authenticated GitHub request; distinguish only explicit 404."""
    remaining = DEADLINE - time.monotonic()
    if remaining <= 0:
        raise Refused('Environment operation deadline reached; inspect before retry')
    command = ['gh', 'api', '--hostname', 'github.com', '--include', '--method', method,
               '-H', 'X-GitHub-Api-Version: 2026-03-10', path]
    if data is not None:
        command += ['--input', '-']
    try:
        result = subprocess.run(command, input=None if data is None else json.dumps(data),
                                capture_output=True, text=True, timeout=min(20, remaining))
        headers, delimiter, body = result.stdout.replace('\r\n', '\n').partition('\n\n')
        status = re.fullmatch(r'HTTP/[0-9.]+ ([0-9]{3})(?: .*)?', headers.split('\n')[0])
        if not delimiter or not status:
            raise Refused('GitHub status unavailable; no absence inferred')
        code = int(status.group(1))
        if code == 404 and allow_absent and method == 'GET':
            return None
        if result.returncode or not 200 <= code < 300:
            raise Refused('GitHub environment request refused; inspect before retry')
        value = json.loads(body)
        if not isinstance(value, dict):
            raise Refused('GitHub environment response is not an object')
        return value
    except (OSError, subprocess.TimeoutExpired, ValueError) as error:
        raise Refused('GitHub environment result unavailable; inspect before retry') from error


def environment(value):
    """Require the exact environment and deny broad or protected-branch defaults."""
    if (not isinstance(value, dict) or value.get('name') != ENVIRONMENT
            or type(value.get('id')) is not int or value['id'] <= 0
            or value.get('deployment_branch_policy') != POLICY):
        raise Refused('Existing environment identity/policy differs; no overwrite')
    return value


def branches():
    """Accept only an empty custom-policy set or one exact branch-only main rule."""
    value = api(ENV + '/deployment-branch-policies?per_page=100')
    rows = value.get('branch_policies')
    count = value.get('total_count')
    if (type(count) is not int or count not in (0, 1) or not isinstance(rows, list)
            or len(rows) != count or any(not isinstance(row, dict) for row in rows)):
        raise Refused('Environment branch inventory is incomplete or broader than profile')
    if rows and (rows[0].get('name') != 'main' or rows[0].get('type') != 'branch'
                 or type(rows[0].get('id')) is not int or rows[0]['id'] <= 0):
        raise Refused('Environment has a foreign branch/tag policy; no overwrite')
    return rows


def reconcile(apply=False):
    """Create deny-by-default custom policy before admitting main; never alter secrets."""
    repository = api(ROOT)
    if (repository.get('id') != REPOSITORY_ID or repository.get('full_name') != REPOSITORY
            or repository.get('default_branch') != 'main'
            or repository.get('permissions', {}).get('admin') is not True):
        raise Refused('Expected repository/admin identity not confirmed')
    current = api(ENV, allow_absent=True)
    if current is None:
        if not apply:
            return {'action': 'create-main-only-environment', 'effect': 'none'}
        current = api(ENV, 'PUT', {'deployment_branch_policy': POLICY})
    environment(current)
    rows = branches()
    if not rows:
        if not apply:
            return {'action': 'add-exact-main-branch-rule', 'effect': 'none'}
        api(ENV + '/deployment-branch-policies', 'POST', {'name': 'main', 'type': 'branch'})
    confirmed = environment(api(ENV)) if apply else current
    if apply and not branches():
        raise Refused('Main branch policy not confirmed; no secret activation')
    return {'environment': ENVIRONMENT, 'environmentID': confirmed['id'],
            'branch': 'main', 'type': 'branch', 'secretsChanged': False,
            'effect': 'verified' if apply else 'none'}


def main():
    """Permit plan or reviewed IaC apply; protected deletion needs separate custody."""
    parser = argparse.ArgumentParser(description=__doc__)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument('--plan', action='store_true')
    mode.add_argument('--apply', action='store_true')
    mode.add_argument('--refuse-destroy', action='store_true')
    args = parser.parse_args()
    try:
        if args.refuse_destroy:
            raise Refused('Environment removal requires separate reviewed custody')
        print(json.dumps(reconcile(args.apply)))
    except Refused as error:
        print('BLOCKED: ' + str(error), file=sys.stderr)
        return 1
    except Exception:
        print('BLOCKED: Environment result unconfirmed; inspect before retry', file=sys.stderr)
        return 1
    return 0


if __name__ == '__main__':
    sys.exit(main())
