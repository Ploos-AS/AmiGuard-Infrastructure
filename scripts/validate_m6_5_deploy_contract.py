#!/usr/bin/env python3
"""Validate the production deploy contract for multi-platform AmiGuard intake."""

from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
QUADLET = ROOT / "deploy/quadlet/amiguard-submit.container"
CADDY = ROOT / "deploy/caddy/Caddyfile.example"
RUNBOOK = ROOT / "docs/M6_5_PRODUCTION_DEPLOYMENT_QUALIFICATION.md"


def require(text: str, needle: str, source: str) -> None:
    if needle not in text:
        raise SystemExit(f"missing required deploy contract in {source}: {needle}")


def forbid(text: str, needle: str, source: str) -> None:
    if needle in text:
        raise SystemExit(f"forbidden deploy contract in {source}: {needle}")


def main() -> int:
    quadlet = QUADLET.read_text(encoding="utf-8")
    caddy = CADDY.read_text(encoding="utf-8")
    runbook = RUNBOOK.read_text(encoding="utf-8")

    # Public listener must remain loopback-only behind Caddy.
    require(quadlet, "PublishPort=127.0.0.1:8080:8080", str(QUADLET))
    require(caddy, "reverse_proxy 127.0.0.1:8080", str(CADDY))

    # Uploads are fail-closed in the checked-in unit and enabled only as an
    # explicit production rollout action after smoke/backup checks.
    require(quadlet, "Environment=AMIGUARD_UPLOAD_ENABLED=false", str(QUADLET))

    # The persistent quarantine mount must keep the same host root so existing
    # schema-v1 records remain readable by the backward-compatible admin CLI.
    require(quadlet, "Volume=/data/amiguard/quarantine:/data/quarantine:rw", str(QUADLET))
    require(quadlet, "Environment=AMIGUARD_QUARANTINE_ROOT=/data/quarantine", str(QUADLET))

    # Runtime hardening must survive the multi-platform rollout.
    for item in ("ReadOnly=true", "NoNewPrivileges=true", "DropCapability=all", "PidsLimit=64"):
        require(quadlet, item, str(QUADLET))

    # Reverse proxy security headers stay part of the production boundary.
    for item in (
        "X-Content-Type-Options nosniff",
        "X-Frame-Options DENY",
        "Referrer-Policy no-referrer",
        "Content-Security-Policy",
    ):
        require(caddy, item, str(CADDY))

    # Runbook explicitly forbids implicit mutation of legacy production data.
    for item in (
        "Do not move, rewrite, or delete existing root-level schema-v1 submissions",
        "AMIGUARD_UPLOAD_ENABLED=false",
        "amiga",
        "atari-st",
        "mac68k",
        ".incoming",
        "rollback",
    ):
        require(runbook, item, str(RUNBOOK))

    forbid(runbook, "automatic migration of schema-v1", str(RUNBOOK))
    print("M6.5 production deployment contract: PASS")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
