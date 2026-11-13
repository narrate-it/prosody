#!/usr/bin/env python3
"""Assemble the Nanos custody volume staging directory from operator custody.

Secrets never enter the app image: this staging tree is written only to the
dedicated prosody-custody TFS volume (ops volume create -d), uploaded as a
separate private image and attached as a data disk. The app image carries only
the empty /custody mountpoint and non-secret env paths.

Transformations:
- installation.json storeDir is rewritten to the in-unikernel volume path.
- Only the prosody-ses AWS profile is copied (config + credentials).
- cursor.key is generated once in operator custody and carried, so directory
  cursor authority survives volume replacement.
- ses.json is carried only when the verified-event subsystem is provisioned.
"""
import configparser
import json
import os
import secrets
import shutil
import sys

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
PRIVATE = os.path.join(REPO, ".private")
STAGE = os.path.join(REPO, "build", "custody")
VOLUME_STORE_DIR = "/custody/keys/store"


def stage_file(src, dst):
    os.makedirs(os.path.dirname(dst), exist_ok=True)
    shutil.copyfile(src, dst)
    os.chmod(dst, 0o600)


def main():
    if not os.path.isdir(PRIVATE):
        sys.exit("operator custody .private/ absent")
    if os.path.exists(STAGE):
        shutil.rmtree(STAGE)
    os.makedirs(STAGE, mode=0o700)

    # Database custody: only the three files the host reads.
    inst = json.load(open(os.path.join(PRIVATE, "db", "installation.json")))
    inst["storeDir"] = VOLUME_STORE_DIR
    dst = os.path.join(STAGE, "db", "installation.json")
    os.makedirs(os.path.dirname(dst), exist_ok=True)
    with open(dst, "w") as f:
        json.dump(inst, f, indent=1)
    os.chmod(dst, 0o600)
    for name in ("roles.json", "ssl.json"):
        stage_file(os.path.join(PRIVATE, "db", name), os.path.join(STAGE, "db", name))

    # Native key store (owner custody).
    shutil.copytree(os.path.join(PRIVATE, "keys", "store"), os.path.join(STAGE, "keys", "store"))
    for root, dirs, files in os.walk(os.path.join(STAGE, "keys")):
        for d in dirs:
            os.chmod(os.path.join(root, d), 0o700)
        for f in files:
            os.chmod(os.path.join(root, f), 0o600)

    # Public TLS custody.
    for name in ("cert.pem", "key.pem"):
        stage_file(os.path.join(PRIVATE, "tls", name), os.path.join(STAGE, "tls", name))

    # Directory cursor key: generate once, carry forever.
    cursor = os.path.join(PRIVATE, "cursor.key")
    if not os.path.exists(cursor):
        with open(cursor, "wb") as f:
            f.write(secrets.token_bytes(32))
        os.chmod(cursor, 0o600)
    stage_file(cursor, os.path.join(STAGE, "cursor.key"))

    # SES verified-event custody only when provisioned.
    ses = os.path.join(PRIVATE, "ses.json")
    if os.path.exists(ses):
        stage_file(ses, os.path.join(STAGE, "ses.json"))
    else:
        print("note: ses.json absent; host will refuse startup until the verified-event gate closes")

    # AWS profile: only prosody-ses, only the fields the SDK reads.
    aws_dir = os.path.join(STAGE, "aws")
    os.makedirs(aws_dir, mode=0o700)
    home = os.path.expanduser("~")
    cfg = configparser.ConfigParser()
    cfg.read(os.path.join(home, ".aws", "config"))
    section = "profile prosody-ses"
    if not cfg.has_section(section):
        sys.exit("prosody-ses profile absent from ~/.aws/config")
    out = configparser.ConfigParser()
    out[section] = {k: v for k, v in cfg.items(section) if k in ("region", "output")}
    with open(os.path.join(aws_dir, "config"), "w") as f:
        out.write(f)
    creds = configparser.ConfigParser()
    creds.read(os.path.join(home, ".aws", "credentials"))
    if not creds.has_section("prosody-ses"):
        sys.exit("prosody-ses credentials absent")
    outc = configparser.ConfigParser()
    outc["prosody-ses"] = dict(creds.items("prosody-ses"))
    with open(os.path.join(aws_dir, "credentials"), "w") as f:
        outc.write(f)
    for name in ("config", "credentials"):
        os.chmod(os.path.join(aws_dir, name), 0o600)

    print("custody staged at", STAGE)


if __name__ == "__main__":
    main()
