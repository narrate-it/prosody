#!/usr/bin/env python3
"""Upload a raw Nanos image to Linode via the real images/upload API.

No object-storage subscription is required: Linode returns a bounded upload
URL for the image. LINODE_TOKEN is read from the environment, never echoed.
Uses curl (system trust store) for all HTTPS.
Usage: linode-image-upload.py <label> <raw-image-path> [region]
"""
import gzip
import json
import os
import subprocess
import sys
import tempfile
import time

API = "https://api.linode.com/v4"


def curl_json(method, url, token, body=None):
    cmd = ["curl", "-sS", "--max-time", "120", "-X", method, "-H", "Authorization: Bearer " + token]
    if body is not None:
        cmd += ["-H", "Content-Type: application/json", "-d", json.dumps(body)]
    cmd.append(url)
    out = subprocess.run(cmd, capture_output=True, text=True, timeout=180)
    if out.returncode != 0:
        sys.exit("curl failed: " + out.stderr[:200])
    return json.loads(out.stdout) if out.stdout.strip() else {}


def main():
    if len(sys.argv) < 3:
        sys.exit(__doc__)
    label, path = sys.argv[1], sys.argv[2]
    region = sys.argv[3] if len(sys.argv) > 3 else "us-west"
    token = os.environ.get("LINODE_TOKEN")
    if not token:
        sys.exit("LINODE_TOKEN absent")
    created = curl_json("POST", API + "/images/upload", token, {
        "label": label, "region": region,
        "description": "Prosody Nanos image (ops-built); custody is a separate private volume image.",
    })
    image = created.get("image", {})
    upload_to = created.get("upload_to")
    if not image.get("id") or not upload_to:
        sys.exit("upload API did not return an image and upload URL: " + json.dumps(created)[:200])
    print("image id:", image["id"], "status:", image.get("status"))
    with open(path, "rb") as f:
        payload = gzip.compress(f.read(), compresslevel=6)
    print("uploading", len(payload), "gzipped bytes")
    with tempfile.NamedTemporaryFile(suffix=".gz", delete=False) as tmp:
        tmp.write(payload)
        tmp_path = tmp.name
    try:
        out = subprocess.run([
            "curl", "-sS", "--max-time", "900", "-X", "PUT",
            "-H", "Content-Type: application/octet-stream",
            "--data-binary", "@" + tmp_path, upload_to,
        ], capture_output=True, text=True, timeout=960)
        if out.returncode != 0:
            sys.exit("upload PUT failed: " + out.stderr[:200])
    finally:
        os.unlink(tmp_path)
    for _ in range(60):
        state = curl_json("GET", API + "/images/" + image["id"], token)
        print("status:", state.get("status"))
        if state.get("status") == "available":
            print("AVAILABLE", state["id"], "size_mb:", state.get("size"))
            return
        if state.get("status") not in ("pending", "pending_upload", "creating", "uploading"):
            sys.exit("unexpected image status: " + str(state.get("status")))
        time.sleep(10)
    sys.exit("image did not become available in time")


if __name__ == "__main__":
    main()
