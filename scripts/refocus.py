#!/usr/bin/env python3
"""Hourly, bounded reminder to the existing implementation session."""
import fcntl
import pathlib
import subprocess
import sys

ROOT = pathlib.Path(__file__).resolve().parents[1]
STATE = ROOT / '.local'
STATE.mkdir(exist_ok=True)
if (ROOT / 'docs/implementation/COMPLETE').exists():
    sys.exit(0)
with (STATE / 'refocus.lock').open('w') as lock:
    try:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
    except BlockingIOError:
        sys.exit(0)
    message = ('Hourly Prosody implementation reminder requested by David: read '
               'docs/implementation/STATUS.md and WORK.md, resume the next unblocked RFC '
               'implementation task, and verify its behavior. Continue beyond AMOS status '
               'updates. Preserve authority/capacity gates; this reminder does not authorize '
               'spend, deployment or bypassing a failed admission. Remove the owned cron '
               'entry when the full authorized implementation and acceptance are complete.')
    try:
        result = subprocess.run(
            [str(pathlib.Path.home() / '.local/bin/codex'), 'queue', '--thread',
             '01a1906b-000a-73d0-ae3c-07cd0f363566', '--message', message, '--resume'],
            cwd=ROOT, capture_output=True, text=True, timeout=45)
        (STATE / 'refocus-last.log').write_text(result.stdout + result.stderr)
        sys.exit(result.returncode)
    except subprocess.TimeoutExpired:
        (STATE / 'refocus-last.log').write_text('Queue outcome unknown after 45 seconds; reconcile before manual retry.\n')
        sys.exit(1)
