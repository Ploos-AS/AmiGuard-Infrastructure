#!/usr/bin/env python3
"""Validate the checked-in M6.8 live production qualification evidence contract."""

from pathlib import Path

DOC = Path("docs/M6_8_LIVE_PRODUCTION_QUALIFICATION.md")

REQUIRED = (
    "**PASS — 2026-09-12**",
    "bf665f712c9592d6a6f9a933e831c89320226507ecb82b5fae9abf3f84291943",
    "c84a016ff8e64d5f293ea48bf97fa19e",
    "6beaed938bfec164fcabd338d706556b0583ce809006fa28d93f36a5b13677f6",
    "39208be41ad40f71958e83072e6f0a12",
    "763d6ab2e01283404d85f566570bba8bce1d9d271484908835426102bf513f9f",
    "256a361b04f9fdf77c0f36092c33e913",
    "1f9b2bb3e1d55bd85a6268114f5b7275f2769c0c2b128467f38ad3a270bbad3f",
    "records=6 verified=6 failed=0 legacy_v1=3 bytes=400",
    "platforms amiga=4 atari-st=1 mac68k=1",
    "2d7e0f325459fa8cd12ec1273d61a342",
    "275a021bbfb6489e54d471899f7db9d1663fc695ec2fe2a2c4538aabf651fd0f",
    "result: ACCEPTED",
    "route_manifest_verified: true",
    "sample_verified: true",
    "analysis_started: false",
    "0400",
    "explicit verified operator action",
)

FORBIDDEN = (
    "automatic transfer of arbitrary public quarantine content into ASW is enabled",
    "automatic signature publication is enabled",
)


def main() -> int:
    if not DOC.is_file():
        raise SystemExit(f"missing {DOC}")
    text = DOC.read_text(encoding="utf-8")
    missing = [item for item in REQUIRED if item not in text]
    forbidden = [item for item in FORBIDDEN if item in text]
    if missing or forbidden:
        if missing:
            print("M6.8 qualification record missing required evidence:")
            for item in missing:
                print(f"  - {item}")
        if forbidden:
            print("M6.8 qualification record contains forbidden claims:")
            for item in forbidden:
                print(f"  - {item}")
        return 1
    print("M6.8 live production qualification record: PASS")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
