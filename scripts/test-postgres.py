#!/usr/bin/env python3
"""An owned, bounded PostgreSQL fixture; no existing cluster is controlled."""
import os
from pathlib import Path
import shutil
import signal
import subprocess
import tempfile
import time

ROOT = Path(__file__).resolve().parents[1]
USER = "prosody_fixture"
PORT = "55439"


def fixture_environment(source):
    # libpq hostaddr/service/user/options can override explicit socket defaults.
    return {key: value for key, value in source.items() if not key.startswith("PG")}


def conninfo_value(value):
    # PostgreSQL keyword/value syntax, not shell escaping. All processes use argv.
    return "'" + str(value).replace("\\", "\\\\").replace("'", "\\'") + "'"


def stop_owned(server):
    """Stop only the exact foreground child we launched, including failed start."""
    if server.poll() is not None:
        if server.returncode != 0:
            raise RuntimeError("Fixture exited unexpectedly; retained data for diagnosis")
        return
    # PostgreSQL interprets SIGINT as fast shutdown (terminate client sessions).
    server.send_signal(signal.SIGINT)
    try:
        server.wait(timeout=30)
    except subprocess.TimeoutExpired:
        # Reap the exact owned child, but retain data and fail the run if graceful
        # shutdown failed. Never issue a signal based on a shared port/PID file.
        server.kill()
        server.wait(timeout=10)
        raise RuntimeError("Fixture shutdown timed out; retained data for diagnosis")
    if server.returncode != 0:
        raise RuntimeError("Fixture shutdown failed; retained data for diagnosis")


def run():
    if os.environ.get("PROSODY_BUILD_ADMITTED") != "1":
        raise RuntimeError("Build lease and capacity admission required")
    state = ROOT / ".local"
    state.mkdir(exist_ok=True)
    fixture = Path(tempfile.mkdtemp(prefix="pg-test.", dir=state))
    data = fixture / "data"
    env = fixture_environment(os.environ)
    server = None
    succeeded = False
    # Keep logs on every outcome; retain the full fixture on any failure.
    with (fixture / "init.log").open("wb") as init_log, (fixture / "server.log").open("wb") as server_log:
        try:
            subprocess.run(
                ["initdb", "-D", str(data), "-U", USER, "--auth-local=trust",
                 "--auth-host=reject", "--no-locale", "-E", "UTF8"],
                env=env, stdout=init_log, stderr=subprocess.STDOUT,
                check=True, timeout=60,
            )
            # Direct foreground ownership avoids pg_ctl's launch/wait gap and
            # its shell interpretation of -o. Spaces/quotes stay in one argument.
            server = subprocess.Popen(
                ["postgres", "-D", str(data), "-c", "listen_addresses=",
                 "-k", str(fixture), "-p", PORT, "-c", "shared_buffers=16MB",
                 "-c", "max_connections=10"],
                env=env, stdout=server_log, stderr=subprocess.STDOUT,
            )
            deadline = time.monotonic() + 60
            while True:
                if server.poll() is not None:
                    raise RuntimeError("Fixture exited during startup")
                ready = subprocess.run(
                    ["pg_isready", "-h", str(fixture), "-p", PORT, "-U", USER,
                     "-d", "postgres", "-t", "1"],
                    env=env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                    timeout=3,
                )
                if ready.returncode == 0:
                    break
                if time.monotonic() >= deadline:
                    raise RuntimeError("Fixture startup timed out")
                time.sleep(0.1)
            subprocess.run(
                ["createdb", "-h", str(fixture), "-p", PORT, "-U", USER,
                 "prosody_test"], env=env, check=True, timeout=15,
            )
            env["PROSODY_TEST_DATABASE_URL"] = (
                f"host={conninfo_value(fixture)} port={PORT} user={USER} "
                "dbname=prosody_test sslmode=disable"
            )
            env["GOMAXPROCS"] = "2"
            env["GOCACHE"] = str(state / "gocache")
            subprocess.run(
                ["go", "test", "-p", "1", "./internal/store", "-count=1", "-timeout=90s"],
                cwd=ROOT, env=env, check=True, timeout=120,
            )
            succeeded = True
        finally:
            try:
                if server is not None:
                    stop_owned(server)
                if succeeded:
                    # Only this invocation's mkdtemp/data, after confirmed shutdown.
                    # No glob, historical-directory pruning or external path input.
                    shutil.rmtree(data)
            finally:
                print(f"PostgreSQL fixture logs: {fixture}")


def interrupted(signum, _frame):
    raise KeyboardInterrupt(f"Interrupted by signal {signum}")


if __name__ == "__main__":
    for signum in (signal.SIGHUP, signal.SIGINT, signal.SIGTERM):
        signal.signal(signum, interrupted)
    run()
