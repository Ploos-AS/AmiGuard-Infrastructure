#!/usr/bin/env python3
from pathlib import Path

quadlet = Path("deploy/quadlet/amiguard-submit.container").read_text(encoding="utf-8")
caddy = Path("deploy/caddy/Caddyfile.example").read_text(encoding="utf-8")
containerfile = Path("submit/Containerfile").read_text(encoding="utf-8")

required_quadlet = [
    "PublishPort=127.0.0.1:8080:8080",
    "ReadOnly=true",
    "NoNewPrivileges=true",
    "DropCapability=all",
    "Memory=192m",
    "PidsLimit=64",
]
for item in required_quadlet:
    if item not in quadlet:
        raise SystemExit(f"M2 hardening error: missing Quadlet setting: {item}")

for forbidden in ("Privileged=true", "Network=host", "/run/podman/podman.sock", "/var/run/docker.sock"):
    if forbidden in quadlet:
        raise SystemExit(f"M2 hardening error: forbidden Quadlet setting: {forbidden}")

if "USER 65532:65532" not in containerfile:
    raise SystemExit("M2 hardening error: container must run as explicit non-root user")

required_headers = [
    "X-Content-Type-Options nosniff",
    "X-Frame-Options DENY",
    "Referrer-Policy no-referrer",
    "Content-Security-Policy",
    "Permissions-Policy",
]
for item in required_headers:
    if item not in caddy:
        raise SystemExit(f"M2 hardening error: missing Caddy security header: {item}")

if "reverse_proxy 127.0.0.1:8080" not in caddy:
    raise SystemExit("M2 hardening error: Caddy must proxy to loopback only")

print("OK: M2 static hardening contract validated")
