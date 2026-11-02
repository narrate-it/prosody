"""Runner control-flow tests: no PostgreSQL, Go, network, or provider processes."""
import importlib.util
from pathlib import Path
import signal
import subprocess
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("fixture_runner", Path(__file__).with_name("test-postgres.py"))
runner = importlib.util.module_from_spec(spec)
spec.loader.exec_module(runner)


class Server:
    def __init__(self):
        self.returncode = None
        self.signals = []

    def poll(self):
        return self.returncode

    def send_signal(self, value):
        self.signals.append(value)

    def wait(self, timeout):
        self.returncode = 0
        return 0


class FixtureRunnerTests(unittest.TestCase):
    def test_client_environment_and_libpq_escaping(self):
        self.assertEqual(runner.fixture_environment({
            "PGHOSTADDR": "elsewhere", "PGUSER": "external", "PGSERVICE": "external",
            "PGOPTIONS": "external", "PGPASSWORD": "secret", "PATH": "kept",
        }), {"PATH": "kept"})
        self.assertEqual(runner.conninfo_value("/space and'quote\\slash"), "'/space and\\'quote\\\\slash'")

    def exercise(self, failure=None):
        with tempfile.TemporaryDirectory(prefix="prosody runner ") as directory:
            root = Path(directory) / "checkout 'with\\characters"
            root.mkdir()
            server = Server()
            calls = []

            def fake_run(argv, **kwargs):
                calls.append(argv)
                self.assertFalse(any(key.startswith("PG") for key in kwargs["env"]))
                if argv[0] == "initdb":
                    data = Path(argv[argv.index("-D") + 1])
                    data.mkdir()
                    (data / "owned-marker").touch()
                if argv[0] == "pg_isready" and failure == "startup":
                    return subprocess.CompletedProcess(argv, 1)
                if argv[0] in ("createdb", "pg_isready"):
                    self.assertEqual(argv[argv.index("-U") + 1], runner.USER)
                    self.assertEqual(argv[argv.index("-p") + 1], runner.PORT)
                if argv[0] == "go":
                    fixture = next((root / ".local").glob("pg-test.*"))
                    self.assertIn("host=" + runner.conninfo_value(fixture), kwargs["env"]["PROSODY_TEST_DATABASE_URL"])
                    if failure == "tests":
                        raise subprocess.CalledProcessError(1, argv)
                return subprocess.CompletedProcess(argv, 0)

            def fake_popen(argv, **kwargs):
                self.assertEqual(argv[0], "postgres")
                self.assertEqual(Path(argv[argv.index("-k") + 1]).parent, root / ".local")
                self.assertFalse(any(key.startswith("PG") for key in kwargs["env"]))
                return server

            with patch.object(runner, "ROOT", root), patch.dict(runner.os.environ, {
                "PROSODY_BUILD_ADMITTED": "1", "PGHOSTADDR": "external",
                "PGUSER": "external", "PGSERVICE": "external",
            }), patch.object(runner.subprocess, "run", side_effect=fake_run), patch.object(runner.subprocess, "Popen", side_effect=fake_popen), patch.object(runner.time, "monotonic", side_effect=[0, 61]):
                if failure == "startup":
                    with self.assertRaisesRegex(RuntimeError, "startup timed out"):
                        runner.run()
                elif failure == "tests":
                    with self.assertRaises(subprocess.CalledProcessError):
                        runner.run()
                else:
                    runner.run()
            fixture = next((root / ".local").glob("pg-test.*"))
            self.assertEqual(server.signals, [signal.SIGINT])
            self.assertEqual((fixture / "data").exists(), failure is not None)
            self.assertTrue((fixture / "init.log").exists())
            self.assertTrue((fixture / "server.log").exists())
            if failure == "startup":
                self.assertFalse(any(argv[0] in ("createdb", "go") for argv in calls))

    def test_success_removes_only_owned_data_preserving_logs(self):
        self.exercise()

    def test_failed_start_is_stopped_and_preserved(self):
        self.exercise("startup")

    def test_failed_tests_preserve_data(self):
        self.exercise("tests")


if __name__ == "__main__":
    unittest.main()
