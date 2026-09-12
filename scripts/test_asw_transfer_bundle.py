from __future__ import annotations

import hashlib
import json
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import asw_transfer_bundle as transfer  # noqa: E402


class ASWTransferBundleTests(unittest.TestCase):
    def make_route(self, root: Path, platform: str, namespace: str, sid: str, payload: bytes) -> None:
        ns = root / namespace
        ns.mkdir(parents=True, exist_ok=True)
        digest = hashlib.sha256(payload).hexdigest()
        (ns / f"{sid}.sample").write_bytes(payload)
        route = {
            "schema_version": 1,
            "kind": "amiguard-asw-routing-manifest",
            "submission_id": sid,
            "platform": platform,
            "asw_namespace": namespace,
            "sha256": digest,
            "size": len(payload),
            "received_at": "2026-09-12T10:00:00Z",
            "routed_at": "2026-09-12T10:05:00Z",
            "source": "amiguard-public-quarantine",
        }
        (ns / f"{sid}.route.json").write_text(json.dumps(route) + "\n", encoding="utf-8")

    def test_prepare_and_verify_all_platforms(self):
        with tempfile.TemporaryDirectory() as tmp:
            base = Path(tmp)
            handoff = base / "handoff"
            handoff.mkdir()
            self.make_route(handoff, "amiga", "amiga", "1" * 32, b"amiga")
            self.make_route(handoff, "atari-st", "atari", "2" * 32, b"atari")
            self.make_route(handoff, "mac68k", "mac68k", "3" * 32, b"mac")
            bundle = base / "bundle"
            manifest = transfer.prepare(handoff, bundle)
            self.assertEqual(len(manifest["entries"]), 3)
            self.assertFalse(manifest["analysis_started"])
            self.assertFalse(manifest["network_transfer_performed"])
            verified = transfer.verify(bundle)
            self.assertEqual(len(verified["entries"]), 3)
            for entry in verified["entries"]:
                sample = bundle / entry["sample_path"]
                route = bundle / entry["route_manifest_path"]
                self.assertEqual(sample.stat().st_mode & 0o777, 0o400)
                self.assertEqual(route.stat().st_mode & 0o777, 0o400)

    def test_tampered_sample_fails_closed(self):
        with tempfile.TemporaryDirectory() as tmp:
            base = Path(tmp)
            handoff = base / "handoff"
            handoff.mkdir()
            sid = "a" * 32
            self.make_route(handoff, "amiga", "amiga", sid, b"good")
            bundle = base / "bundle"
            transfer.prepare(handoff, bundle)
            sample = bundle / "amiga" / f"{sid}.sample"
            sample.chmod(0o600)
            sample.write_bytes(b"tampered")
            with self.assertRaisesRegex(ValueError, "hash/size|verification"):
                transfer.verify(bundle)

    def test_symlink_route_is_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            base = Path(tmp)
            handoff = base / "handoff"
            handoff.mkdir()
            sid = "b" * 32
            self.make_route(handoff, "mac68k", "mac68k", sid, b"fixture")
            route = handoff / "mac68k" / f"{sid}.route.json"
            target = handoff / "mac68k" / "target.json"
            route.rename(target)
            route.symlink_to(target.name)
            with self.assertRaisesRegex(ValueError, "non-symlink"):
                transfer.prepare(handoff, base / "bundle")


if __name__ == "__main__":
    unittest.main()
