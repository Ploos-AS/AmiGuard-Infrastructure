#!/usr/bin/env python3
from pathlib import Path

quadlet = Path("deploy/quadlet/amiguard-submit.container").read_text(encoding="utf-8")

required = [
    "PublishPort=127.0.0.1:8080:8080",
    "ReadOnly=true",
    "NoNewPrivileges=true",
    "DropCapability=all",
    "UserNS=keep-id:uid=65532,gid=65532",
    "Environment=AMIGUARD_UPLOAD_ENABLED=false",
    "Environment=AMIGUARD_QUARANTINE_ROOT=/data/quarantine",
    "Environment=AMIGUARD_MAX_UPLOAD_BYTES=16777216",
    "Volume=/data/amiguard/quarantine:/data/quarantine:rw",
    "PidsLimit=64",
    "MemoryMax=192M",
]

for item in required:
    if item not in quadlet:
        raise SystemExit(f"M4 closed VPS contract error: missing setting: {item}")

for forbidden in (
    "PublishPort=0.0.0.0:8080:8080",
    "Environment=AMIGUARD_UPLOAD_ENABLED=true",
    "Privileged=true",
    "Network=host",
):
    if forbidden in quadlet:
        raise SystemExit(f"M4 closed VPS contract error: forbidden setting: {forbidden}")

print("OK: M4 closed VPS deployment contract validated")
