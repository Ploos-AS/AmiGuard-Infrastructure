#!/usr/bin/env python3
from __future__ import annotations

import hashlib
import json
import tempfile
import unittest
from pathlib import Path

import inventory_quarantine


def write_record(root: Path, submission_id: str, payload: bytes, *, platform: str | None = None) -> Path:
    directory = root if platform is None else root / platform
    directory.mkdir(parents=True, exist_ok=True)
    digest = hashlib.sha256(payload).hexdigest()
    metadata = {
        "schema_version": 1 if platform is None else 2,
        "kind": "amiguard-quarantine-submission",
        "submission_id": submission_id,
        "sha256": digest,
        "size": len(payload),
        "received_at": "2026-09-12T10:00:00Z",
        "consent": True,
        "executed": False,
        "extracted": False,
    }
    if platform is not None:
        metadata["platform"] = platform
    (directory / f"{submission_id}.sample").write_bytes(payload)
    (directory / f"{submission_id}.json").write_text(json.dumps(metadata), encoding="utf-8")
    return directory


class InventoryTests(unittest.TestCase):
    def test_valid_legacy_and_namespaced_records(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_record(root, "1" * 32, b"legacy-amiga\n")
            write_record(root, "2" * 32, b"new-amiga\n", platform="amiga")
            write_record(root, "3" * 32, b"new-atari\n", platform="atari-st")
            write_record(root, "4" * 32, b"new-mac\n", platform="mac68k")
            report = inventory_quarantine.inventory(root)
            self.assertTrue(report["ok"])
            self.assertEqual(report["records"], 4)
            self.assertEqual(report["verified_records"], 4)
            self.assertEqual(report["legacy_schema_v1"], 1)
            self.assertEqual(report["platform_counts"], {"amiga": 2, "atari-st": 1, "mac68k": 1})

    def test_orphan_sample_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            submission_id = "a" * 32
            (root / f"{submission_id}.sample").write_bytes(b"orphan")
            report = inventory_quarantine.inventory(root)
            self.assertFalse(report["ok"])
            self.assertEqual(report["failed_records"], 1)
            self.assertIn("metadata", " ".join(report["record_results"][0]["issues"]))

    def test_tampered_sample_fails_hash_and_size(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            submission_id = "b" * 32
            directory = write_record(root, submission_id, b"before\n", platform="atari-st")
            (directory / f"{submission_id}.sample").write_bytes(b"after-tamper\n")
            report = inventory_quarantine.inventory(root)
            self.assertFalse(report["ok"])
            issues = " ".join(report["record_results"][0]["issues"])
            self.assertIn("sha256 mismatch", issues)
            self.assertIn("size mismatch", issues)

    def test_duplicate_legacy_and_namespace_id_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            submission_id = "c" * 32
            write_record(root, submission_id, b"legacy\n")
            write_record(root, submission_id, b"v2\n", platform="amiga")
            report = inventory_quarantine.inventory(root)
            self.assertFalse(report["ok"])
            self.assertTrue(any("duplicate submission id" in issue for issue in report["issues"]))

    def test_nonempty_incoming_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            incoming = root / ".incoming"
            incoming.mkdir()
            (incoming / ".incoming-test").write_bytes(b"staged")
            report = inventory_quarantine.inventory(root)
            self.assertFalse(report["ok"])
            self.assertEqual(report["incoming_entries"], [".incoming-test"])

    def test_symlinked_sample_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            submission_id = "d" * 32
            directory = write_record(root, submission_id, b"payload\n", platform="mac68k")
            sample = directory / f"{submission_id}.sample"
            sample.unlink()
            target = root / "target"
            target.write_bytes(b"payload\n")
            sample.symlink_to(target)
            report = inventory_quarantine.inventory(root)
            self.assertFalse(report["ok"])
            self.assertIn("symlink", " ".join(report["record_results"][0]["issues"]))


if __name__ == "__main__":
    unittest.main()
