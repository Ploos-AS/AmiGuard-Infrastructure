#!/usr/bin/env python3
"""Read-only AmiGuard quarantine inventory and integrity verification.

The scanner never creates, renames, rewrites, or removes quarantine content.
It understands legacy schema-v1 records at the quarantine root and schema-v2
records below the supported public platform namespaces.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import stat
import sys
from dataclasses import dataclass, asdict
from pathlib import Path
from typing import Any

ID_RE = re.compile(r"^[0-9a-f]{32}$")
SHA_RE = re.compile(r"^[0-9a-f]{64}$")
PLATFORMS = ("amiga", "atari-st", "mac68k")
KIND = "amiguard-quarantine-submission"


@dataclass
class RecordResult:
    submission_id: str
    platform: str
    schema_version: int
    metadata_path: str
    sample_path: str
    size: int | None = None
    sha256: str | None = None
    ok: bool = False
    issues: list[str] | None = None

    def __post_init__(self) -> None:
        if self.issues is None:
            self.issues = []


def regular_nonsymlink(path: Path) -> bool:
    try:
        mode = path.lstat().st_mode
    except FileNotFoundError:
        return False
    return stat.S_ISREG(mode) and not stat.S_ISLNK(mode)


def directory_nonsymlink(path: Path) -> bool:
    try:
        mode = path.lstat().st_mode
    except FileNotFoundError:
        return False
    return stat.S_ISDIR(mode) and not stat.S_ISLNK(mode)


def hash_file(path: Path) -> tuple[str, int]:
    h = hashlib.sha256()
    size = 0
    with path.open("rb") as fh:
        while True:
            chunk = fh.read(1024 * 1024)
            if not chunk:
                break
            h.update(chunk)
            size += len(chunk)
    return h.hexdigest(), size


def load_metadata(path: Path) -> dict[str, Any]:
    if not regular_nonsymlink(path):
        raise ValueError("metadata is missing, non-regular, or a symlink")
    if path.stat().st_size > 64 * 1024:
        raise ValueError("metadata exceeds 64 KiB")
    with path.open("r", encoding="utf-8") as fh:
        value = json.load(fh)
    if not isinstance(value, dict):
        raise ValueError("metadata must be a JSON object")
    return value


def validate_record(root: Path, directory: Path, submission_id: str, platform: str, schema: int) -> RecordResult:
    metadata_path = directory / f"{submission_id}.json"
    sample_path = directory / f"{submission_id}.sample"
    result = RecordResult(
        submission_id=submission_id,
        platform=platform,
        schema_version=schema,
        metadata_path=str(metadata_path.relative_to(root)),
        sample_path=str(sample_path.relative_to(root)),
    )

    try:
        metadata = load_metadata(metadata_path)
    except (OSError, ValueError, json.JSONDecodeError, UnicodeDecodeError) as exc:
        result.issues.append(f"metadata: {exc}")
        return result

    expected_keys_v1 = {
        "schema_version", "kind", "submission_id", "sha256", "size",
        "received_at", "consent", "executed", "extracted",
    }
    expected_keys_v2 = expected_keys_v1 | {"platform"}
    expected_keys = expected_keys_v1 if schema == 1 else expected_keys_v2
    extras = set(metadata) - expected_keys
    missing = expected_keys - set(metadata)
    if extras:
        result.issues.append("unexpected metadata fields: " + ",".join(sorted(extras)))
    if missing:
        result.issues.append("missing metadata fields: " + ",".join(sorted(missing)))

    if metadata.get("schema_version") != schema:
        result.issues.append("schema_version mismatch")
    if metadata.get("kind") != KIND:
        result.issues.append("kind mismatch")
    if metadata.get("submission_id") != submission_id:
        result.issues.append("submission_id mismatch")
    if schema == 2 and metadata.get("platform") != platform:
        result.issues.append("platform mismatch")
    if metadata.get("consent") is not True:
        result.issues.append("consent is not true")
    if metadata.get("executed") is not False:
        result.issues.append("executed is not false")
    if metadata.get("extracted") is not False:
        result.issues.append("extracted is not false")

    expected_sha = metadata.get("sha256")
    expected_size = metadata.get("size")
    if not isinstance(expected_sha, str) or not SHA_RE.fullmatch(expected_sha):
        result.issues.append("invalid metadata sha256")
    if not isinstance(expected_size, int) or isinstance(expected_size, bool) or expected_size < 0:
        result.issues.append("invalid metadata size")

    if not regular_nonsymlink(sample_path):
        result.issues.append("sample is missing, non-regular, or a symlink")
        return result

    try:
        digest, size = hash_file(sample_path)
    except OSError as exc:
        result.issues.append(f"sample read: {exc}")
        return result
    result.sha256 = digest
    result.size = size
    if isinstance(expected_sha, str) and SHA_RE.fullmatch(expected_sha) and digest != expected_sha:
        result.issues.append("sample sha256 mismatch")
    if isinstance(expected_size, int) and not isinstance(expected_size, bool) and size != expected_size:
        result.issues.append("sample size mismatch")
    result.ok = not result.issues
    return result


def candidate_ids(directory: Path) -> tuple[set[str], list[str]]:
    ids: set[str] = set()
    issues: list[str] = []
    for entry in os.scandir(directory):
        name = entry.name
        if name.startswith("."):
            continue
        if name.endswith(".json") or name.endswith(".sample"):
            submission_id = name.rsplit(".", 1)[0]
            if ID_RE.fullmatch(submission_id):
                ids.add(submission_id)
            else:
                issues.append(f"unexpected quarantine filename: {name}")
    return ids, issues


def inventory(root: Path) -> dict[str, Any]:
    root = root.resolve(strict=True)
    if not directory_nonsymlink(root):
        raise ValueError("quarantine root must be a non-symlink directory")

    records: list[RecordResult] = []
    issues: list[str] = []
    seen: dict[str, str] = {}

    legacy_ids, legacy_issues = candidate_ids(root)
    issues.extend(legacy_issues)
    for submission_id in sorted(legacy_ids):
        location = "legacy"
        if submission_id in seen:
            issues.append(f"duplicate submission id {submission_id}: {seen[submission_id]} and {location}")
        else:
            seen[submission_id] = location
        records.append(validate_record(root, root, submission_id, "amiga", 1))

    for platform in PLATFORMS:
        directory = root / platform
        if not directory.exists():
            continue
        if not directory_nonsymlink(directory):
            issues.append(f"platform namespace {platform} is not a non-symlink directory")
            continue
        ids, namespace_issues = candidate_ids(directory)
        issues.extend(f"{platform}: {issue}" for issue in namespace_issues)
        for submission_id in sorted(ids):
            location = platform
            if submission_id in seen:
                issues.append(f"duplicate submission id {submission_id}: {seen[submission_id]} and {location}")
            else:
                seen[submission_id] = location
            records.append(validate_record(root, directory, submission_id, platform, 2))

    incoming = root / ".incoming"
    incoming_files: list[str] = []
    if incoming.exists():
        if not directory_nonsymlink(incoming):
            issues.append(".incoming is not a non-symlink directory")
        else:
            incoming_files = sorted(entry.name for entry in os.scandir(incoming))
            if incoming_files:
                issues.append(f".incoming is not empty ({len(incoming_files)} entries)")

    bad_records = [record for record in records if not record.ok]
    total_bytes = sum(record.size or 0 for record in records if record.ok)
    counts = {platform: 0 for platform in PLATFORMS}
    legacy_count = 0
    for record in records:
        counts[record.platform] = counts.get(record.platform, 0) + 1
        if record.schema_version == 1:
            legacy_count += 1

    return {
        "kind": "amiguard-quarantine-inventory",
        "schema_version": 1,
        "read_only": True,
        "root": str(root),
        "records": len(records),
        "verified_records": len(records) - len(bad_records),
        "failed_records": len(bad_records),
        "legacy_schema_v1": legacy_count,
        "platform_counts": counts,
        "verified_total_bytes": total_bytes,
        "incoming_entries": incoming_files,
        "issues": issues,
        "record_results": [asdict(record) for record in records],
        "ok": not issues and not bad_records,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="Read-only AmiGuard quarantine inventory")
    parser.add_argument("root", type=Path, help="quarantine root")
    parser.add_argument("--json", action="store_true", help="emit complete JSON report")
    args = parser.parse_args()
    try:
        report = inventory(args.root)
    except (OSError, ValueError) as exc:
        print(f"inventory failed: {exc}", file=sys.stderr)
        return 2

    if args.json:
        json.dump(report, sys.stdout, indent=2, sort_keys=True)
        print()
    else:
        print("AmiGuard quarantine inventory (READ ONLY)")
        print(f"root={report['root']}")
        print(
            "records={records} verified={verified_records} failed={failed_records} "
            "legacy_v1={legacy_schema_v1} bytes={verified_total_bytes}".format(**report)
        )
        counts = report["platform_counts"]
        print(f"platforms amiga={counts.get('amiga', 0)} atari-st={counts.get('atari-st', 0)} mac68k={counts.get('mac68k', 0)}")
        if report["issues"]:
            for issue in report["issues"]:
                print(f"ISSUE {issue}")
        for record in report["record_results"]:
            if not record["ok"]:
                print(f"FAIL {record['submission_id']} platform={record['platform']} issues={' | '.join(record['issues'])}")
        print("PASS" if report["ok"] else "FAIL")
    return 0 if report["ok"] else 1


if __name__ == "__main__":
    raise SystemExit(main())
