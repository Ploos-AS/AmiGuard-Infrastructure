#!/usr/bin/env python3
"""Validate the checked-in N100 ASW readiness contract without requiring hardware."""

from pathlib import Path

DOC = Path("docs/M6_10_N100_READINESS.md")
REQUIRED = (
    "M6.10 — Dedicated N100 ASW readiness",
    "LIVE QUALIFICATION DEFERRED",
    "analysis_started=false",
    "amiga",
    "atari",
    "mac68k",
    "asw_transfer_bundle.py verify",
    "asw_route_import.py",
    "fail closed",
    "must not be declared PASS",
)
FORBIDDEN = (
    "M6.10 live qualification: PASS",
    "N100 qualification is PASS",
)


def main() -> int:
    if not DOC.is_file():
        raise SystemExit(f"missing {DOC}")
    text = DOC.read_text(encoding="utf-8")
    missing = [x for x in REQUIRED if x not in text]
    forbidden = [x for x in FORBIDDEN if x in text]
    if missing or forbidden:
        for x in missing:
            print(f"missing required M6.10 readiness statement: {x}")
        for x in forbidden:
            print(f"forbidden premature PASS claim: {x}")
        return 1
    print("M6.10 N100 readiness contract: PASS (live qualification deferred)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
