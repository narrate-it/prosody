#!/usr/bin/env python3
"""Hermetic environment-policy controls without any GitHub or secret effects."""
import importlib.util
import json
from pathlib import Path
import subprocess
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('env_setup', Path(__file__).with_name('configure-asc-environment.py'))
env = importlib.util.module_from_spec(spec)
spec.loader.exec_module(env)


class Policy(unittest.TestCase):
    """Check creation ordering and fail-closed handling of existing policy."""

    def setUp(self):
        """Install a synthetic repository/environment API with recorded writes."""
        self.current = None
        self.rows = []
        self.writes = []
        self.deny_create = False
        self.repository = {'id': env.REPOSITORY_ID, 'full_name': env.REPOSITORY,
                           'default_branch': 'main', 'permissions': {'admin': True}}
        self.patcher = patch.object(env, 'api', side_effect=self.api)
        self.patcher.start()
        self.addCleanup(self.patcher.stop)

    def api(self, path, method='GET', data=None, allow_absent=False):
        """Model only the fixed repository, environment and branch-policy endpoints."""
        if method != 'GET':
            self.writes.append((path, method, data))
        if path == env.ROOT:
            return self.repository
        if path == env.ENV:
            if method == 'PUT':
                self.current = {'id': 123, 'name': env.ENVIRONMENT,
                                'deployment_branch_policy': data['deployment_branch_policy']}
            return self.current
        if path == env.ENV + '/deployment-branch-policies?per_page=100':
            return {'total_count': len(self.rows), 'branch_policies': self.rows}
        if path == env.ENV + '/deployment-branch-policies' and method == 'POST':
            self.assertEqual(self.current['deployment_branch_policy'], env.POLICY)
            if self.deny_create:
                raise env.Refused('Synthetic branch-policy failure')
            self.rows = [{'id': 456, **data}]
            return self.rows[0]
        self.fail('Unexpected synthetic endpoint')

    def test_plan_has_no_writes(self):
        """Absent environment planning performs no creation or token operation."""
        self.assertEqual(env.reconcile()['effect'], 'none')
        self.assertEqual(self.writes, [])

    def test_create_then_verify_no_secret_or_duplicate_writes(self):
        """Enable custom policy before adding exactly main, then only read on reuse."""
        self.assertEqual(env.reconcile(True)['branch'], 'main')
        self.assertEqual(self.writes, [(env.ENV, 'PUT', {'deployment_branch_policy': env.POLICY}),
                                      (env.ENV + '/deployment-branch-policies', 'POST', {'name': 'main', 'type': 'branch'})])
        before = list(self.writes)
        self.assertEqual(env.reconcile(True)['secretsChanged'], False)
        self.assertEqual(self.writes, before)

    def test_wrong_repository_or_admin_denied(self):
        """An unrelated repository or missing administrator authority cannot be changed."""
        for field, value in (('id', 1), ('full_name', 'other/repo'), ('default_branch', 'other'),
                             ('permissions', {'admin': False})):
            original = dict(self.repository)
            with self.subTest(field=field):
                self.repository[field] = value
                with self.assertRaises(env.Refused):
                    env.reconcile(True)
                self.assertEqual(self.writes, [])
            self.repository = original

    def test_existing_broad_policy_never_overwritten(self):
        """An unknown or broader existing environment refuses all write operations."""
        for policy in (None, {'protected_branches': True, 'custom_branch_policies': False},
                       {'protected_branches': False, 'custom_branch_policies': False}):
            self.current = {'id': 123, 'name': env.ENVIRONMENT, 'deployment_branch_policy': policy}
            with self.subTest(policy=policy), self.assertRaises(env.Refused):
                env.reconcile(True)
            self.assertEqual(self.writes, [])

    def test_foreign_branch_tag_and_incomplete_list_refused(self):
        """Main tags, wildcard rules and incomplete inventories do not match main branch."""
        self.current = {'id': 123, 'name': env.ENVIRONMENT, 'deployment_branch_policy': env.POLICY}
        for rows in ([{'id': 1, 'name': 'main', 'type': 'tag'}],
                     [{'id': 1, 'name': '*', 'type': 'branch'}], [None],
                     [{'id': 1, 'name': 'main', 'type': 'branch'}] * 2):
            self.rows = rows
            with self.subTest(rows=rows), self.assertRaises(env.Refused):
                env.reconcile(True)
            self.assertEqual(self.writes, [])

    def test_failed_rule_write_leaves_no_broad_allow_rule(self):
        """A partial failure preserves restricted policy and requires reconciliation."""
        self.deny_create = True
        with self.assertRaises(env.Refused):
            env.reconcile(True)
        self.assertEqual(self.current['deployment_branch_policy'], env.POLICY)
        self.assertEqual(self.rows, [])
        self.deny_create = False
        env.reconcile(True)
        self.assertEqual(sum(method == 'PUT' for _, method, _ in self.writes), 1)
        self.assertEqual(self.rows, [{'id': 456, 'name': 'main', 'type': 'branch'}])


class Transport(unittest.TestCase):
    """Verify finite status handling separately from synthetic policy tests."""

    def response(self, code, body, exit_code=0):
        """Make one captured HTTP-shaped GitHub CLI response."""
        return subprocess.CompletedProcess([], exit_code,
                                           'HTTP/2.0 %d Test\r\nHeader: value\r\n\r\n%s' % (code, json.dumps(body)),
                                           'PRIVATE_DIAGNOSTIC')

    def test_explicit_404_only_absence(self):
        """Only an explicit successful-status parse permits the absent-environment branch."""
        for code, result in ((404, None), (403, 'refuse'), (500, 'refuse')):
            with self.subTest(code=code), patch.object(env.subprocess, 'run', return_value=self.response(code, {}, 1)):
                if result is None:
                    self.assertIsNone(env.api(env.ENV, allow_absent=True))
                else:
                    with self.assertRaises(env.Refused):
                        env.api(env.ENV, allow_absent=True)
        with patch.object(env.subprocess, 'run', return_value=subprocess.CompletedProcess([], 1, '', 'PRIVATE')):
            with self.assertRaises(env.Refused):
                env.api(env.ENV, allow_absent=True)

    def test_request_is_fixed_host_and_bounded(self):
        """JSON request body uses stdin and never includes an authentication token."""
        with patch.object(env.subprocess, 'run', return_value=self.response(200, {'id': 123})) as run:
            self.assertEqual(env.api(env.ENV, 'PUT', {'deployment_branch_policy': env.POLICY}), {'id': 123})
        args = run.call_args.args[0]
        self.assertIn('github.com', args)
        self.assertNotIn('Authorization', ' '.join(args))
        self.assertEqual(json.loads(run.call_args.kwargs['input']), {'deployment_branch_policy': env.POLICY})
        self.assertLessEqual(run.call_args.kwargs['timeout'], 20)


if __name__ == '__main__':
    unittest.main(verbosity=2)
