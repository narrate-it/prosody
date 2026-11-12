#!/usr/bin/env python3
"""One-shot guarded unlock + auto-lock timeout extension for the prosody-ci signing
keychain, so a release build does not hit a mid-build relock.

Uses the owner-supplied PROSODY_KEYCHAIN_PASSWORD environment secret via a bounded
in-memory SecKeychainUnlock buffer (cleared afterwards). The secret is never printed,
logged, or passed in command arguments, and is removed from the environment before any
subprocess. This does not change the default keychain, the search list, or private-key
access; it only extends the dedicated CI keychain's inactivity auto-lock timeout.
"""
import ctypes
import os
import subprocess
import sys
from pathlib import Path

TIMEOUT_SECONDS = '3600'


def main() -> int:
    if (os.environ.get('GITHUB_REPOSITORY') != 'narrate-it/prosody'
            or os.environ.get('GITHUB_REF') != 'refs/heads/main'
            or os.environ.get('RUNNER_OS') != 'macOS'):
        print('BLOCKED: trusted main self-hosted Mac context required', flush=True)
        return 1
    password = os.environ.pop('PROSODY_KEYCHAIN_PASSWORD', '')
    if not password:
        print('BLOCKED: no configured unlock credential', flush=True)
        return 1
    keychain = Path.home() / 'Library/Keychains/prosody-ci.keychain-db'
    if not keychain.is_file():
        print('BLOCKED: prosody-ci keychain absent', flush=True)
        return 1
    security = ctypes.CDLL('/System/Library/Frameworks/Security.framework/Security')
    core = ctypes.CDLL('/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation')
    security.SecKeychainOpen.argtypes = [ctypes.c_char_p, ctypes.POINTER(ctypes.c_void_p)]
    security.SecKeychainOpen.restype = ctypes.c_int32
    security.SecKeychainGetStatus.argtypes = [ctypes.c_void_p, ctypes.POINTER(ctypes.c_uint32)]
    security.SecKeychainGetStatus.restype = ctypes.c_int32
    security.SecKeychainUnlock.argtypes = [ctypes.c_void_p, ctypes.c_uint32, ctypes.c_void_p, ctypes.c_uint8]
    security.SecKeychainUnlock.restype = ctypes.c_int32
    core.CFRelease.argtypes = [ctypes.c_void_p]
    core.CFRelease.restype = None
    reference = ctypes.c_void_p()
    if security.SecKeychainOpen(os.fsencode(str(keychain)), ctypes.byref(reference)) != 0 or not reference.value:
        print('BLOCKED: keychain metadata unavailable', flush=True)
        return 1
    try:
        status = ctypes.c_uint32()
        if security.SecKeychainGetStatus(reference, ctypes.byref(status)) != 0:
            print('BLOCKED: keychain status unavailable', flush=True)
            return 1
        if not status.value & 1:
            raw = password.encode('utf-8')
            if not 1 <= len(raw) <= 4096:
                print('BLOCKED: configured credential invalid; no unlock attempted', flush=True)
                return 1
            buffer = ctypes.create_string_buffer(raw)
            try:
                result = security.SecKeychainUnlock(reference, len(raw), buffer, 1)
            finally:
                ctypes.memset(buffer, 0, ctypes.sizeof(buffer))
            if result != 0:
                print('BLOCKED: keychain unlock failed; authentication retries STOPPED', flush=True)
                return 1
            if security.SecKeychainGetStatus(reference, ctypes.byref(status)) != 0 or not status.value & 1:
                print('BLOCKED: keychain unlock not confirmed; STOPPED', flush=True)
                return 1
        # Extend inactivity auto-lock timeout; no password argument, no lock-on-sleep.
        subprocess.run(['security', 'set-keychain-settings', '-t', TIMEOUT_SECONDS, str(keychain)],
                       check=True, timeout=20)
        print('OK: prosody-ci unlocked; auto-lock timeout set to ' + TIMEOUT_SECONDS + 's', flush=True)
        return 0
    finally:
        core.CFRelease(reference)


if __name__ == '__main__':
    sys.exit(main())
