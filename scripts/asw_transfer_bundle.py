#!/usr/bin/env python3
"""Prepare and verify explicit VPS -> ASW transfer bundles.

This tool never performs network transfer and never starts analysis. It only
validates an existing AmiGuard route handoff, copies it into a self-describing
bundle, and verifies that bundle after transport.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import shutil
import stat
import tempfile
from datetime import datetime, timezone
from pathlib import Path

ID_RE = re.compile(r"^[0-9a-f]{32}$")
SHA_RE = re.compile(r"^[0-9a-f]{64}$")
PLATFORM_TO_NAMESPACE = {"amiga": "amiga", "atari-st": "atari", "mac68k": "mac68k"}
ROUTE_FIELDS = {
    "schema_version", "kind", "submission_id", "platform", "asw_namespace",
    "sha256", "size", "received_at", "routed_at", "source",
}
MAX_JSON = 64 << 10


def utcnow() -> str:
    return datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")


def require_directory(path: Path) -> None:
    st = path.lstat()
    if stat.S_ISLNK(st.st_mode) or not stat.S_ISDIR(st.st_mode):
        raise ValueError(f"{path} must be an existing non-symlink directory")


def require_regular(path: Path) -> None:
    st = path.lstat()
    if stat.S_ISLNK(st.st_mode) or not stat.S_ISREG(st.st_mode):
        raise ValueError(f"{path} must be a regular non-symlink file")


def hash_file(path: Path) -> tuple[str, int]:
    h = hashlib.sha256()
    size = 0
    with path.open("rb", buffering=0) as f:
        while chunk := f.read(1024 * 1024):
            h.update(chunk)
            size += len(chunk)
    return h.hexdigest(), size


def read_json(path: Path) -> dict:
    require_regular(path)
    if path.stat().st_size > MAX_JSON:
        raise ValueError(f"{path} exceeds JSON size limit")
    return json.loads(path.read_text(encoding="utf-8"))


def validate_route(sample: Path, route_path: Path) -> dict:
    require_regular(sample)
    route = read_json(route_path)
    if set(route) != ROUTE_FIELDS:
        raise ValueError("routing manifest fields do not match contract")
    if route["schema_version"] != 1 or route["kind"] != "amiguard-asw-routing-manifest":
        raise ValueError("routing manifest contract mismatch")
    sid = str(route["submission_id"])
    if not ID_RE.fullmatch(sid):
        raise ValueError("invalid submission id")
    platform = route["platform"]
    if platform not in PLATFORM_TO_NAMESPACE:
        raise ValueError("unsupported platform")
    namespace = PLATFORM_TO_NAMESPACE[platform]
    if route["asw_namespace"] != namespace:
        raise ValueError("platform/namespace mismatch")
    if sample.name != f"{sid}.sample" or route_path.name != f"{sid}.route.json":
        raise ValueError("handoff filenames do not match submission id")
    if not SHA_RE.fullmatch(str(route["sha256"])) or not isinstance(route["size"], int) or route["size"] < 0:
        raise ValueError("invalid sample identity")
    if route["source"] != "amiguard-public-quarantine":
        raise ValueError("unexpected routing source")
    digest, size = hash_file(sample)
    if digest != route["sha256"] or size != route["size"]:
        raise ValueError("routed sample hash/size verification failed")
    return route


def collect_routes(handoff_root: Path) -> list[tuple[dict, Path, Path]]:
    require_directory(handoff_root)
    records: list[tuple[dict, Path, Path]] = []
    for namespace in sorted(set(PLATFORM_TO_NAMESPACE.values())):
        ns = handoff_root / namespace
        if not ns.exists():
            continue
        require_directory(ns)
        for route_path in sorted(ns.glob("*.route.json")):
            require_regular(route_path)
            sid = route_path.name.removesuffix(".route.json")
            sample = ns / f"{sid}.sample"
            route = validate_route(sample, route_path)
            if route["asw_namespace"] != namespace:
                raise ValueError("route manifest stored in wrong namespace")
            records.append((route, sample, route_path))
    if not records:
        raise ValueError("handoff contains no verified routes")
    return records


def copy_exclusive(src: Path, dst: Path, mode: int) -> None:
    dst.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
    if dst.exists() or dst.is_symlink():
        raise FileExistsError(f"refusing to overwrite {dst}")
    fd, temp_name = tempfile.mkstemp(prefix="transfer-", dir=dst.parent)
    temp = Path(temp_name)
    try:
        with src.open("rb") as inp, os.fdopen(fd, "wb") as out:
            shutil.copyfileobj(inp, out, 1024 * 1024)
            out.flush()
            os.fsync(out.fileno())
        os.chmod(temp, mode)
        os.link(temp, dst)
    finally:
        temp.unlink(missing_ok=True)


def prepare(handoff_root: Path, bundle_root: Path) -> dict:
    require_directory(handoff_root)
    if bundle_root.exists():
        raise FileExistsError(f"refusing existing bundle root {bundle_root}")
    bundle_root.mkdir(mode=0o700, parents=False)
    entries = []
    for route, sample, route_path in collect_routes(handoff_root):
        namespace = route["asw_namespace"]
        sid = route["submission_id"]
        out_ns = bundle_root / namespace
        out_ns.mkdir(mode=0o700, exist_ok=True)
        sample_out = out_ns / sample.name
        route_out = out_ns / route_path.name
        copy_exclusive(sample, sample_out, 0o400)
        copy_exclusive(route_path, route_out, 0o400)
        digest, size = hash_file(sample_out)
        if digest != route["sha256"] or size != route["size"]:
            raise ValueError("post-copy sample verification failed")
        route_digest, route_size = hash_file(route_out)
        entries.append({
            "submission_id": sid,
            "platform": route["platform"],
            "asw_namespace": namespace,
            "sample_path": f"{namespace}/{sample.name}",
            "route_manifest_path": f"{namespace}/{route_path.name}",
            "sha256": digest,
            "size": size,
            "route_manifest_sha256": route_digest,
            "route_manifest_size": route_size,
            "analysis_started": False,
        })
    manifest = {
        "schema": "amiguard.asw.transfer-bundle/1",
        "source": "amiguard-production-vps",
        "destination_role": "dedicated-asw-workstation",
        "created_at": utcnow(),
        "network_transfer_performed": False,
        "analysis_started": False,
        "entries": entries,
    }
    manifest_path = bundle_root / "transfer-manifest.json"
    manifest_path.write_text(json.dumps(manifest, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    os.chmod(manifest_path, 0o400)
    verify(bundle_root)
    return manifest


def verify(bundle_root: Path) -> dict:
    require_directory(bundle_root)
    manifest_path = bundle_root / "transfer-manifest.json"
    manifest = read_json(manifest_path)
    expected = {"schema", "source", "destination_role", "created_at", "network_transfer_performed", "analysis_started", "entries"}
    if set(manifest) != expected:
        raise ValueError("transfer manifest fields do not match contract")
    if manifest["schema"] != "amiguard.asw.transfer-bundle/1":
        raise ValueError("transfer manifest schema mismatch")
    if manifest["source"] != "amiguard-production-vps" or manifest["destination_role"] != "dedicated-asw-workstation":
        raise ValueError("transfer endpoint role mismatch")
    if manifest["network_transfer_performed"] is not False or manifest["analysis_started"] is not False:
        raise ValueError("transfer bundle violates explicit operator boundary")
    if not isinstance(manifest["entries"], list) or not manifest["entries"]:
        raise ValueError("transfer bundle has no entries")
    seen: set[str] = set()
    for entry in manifest["entries"]:
        sid = str(entry.get("submission_id", ""))
        if not ID_RE.fullmatch(sid) or sid in seen:
            raise ValueError("invalid or duplicate transfer submission id")
        seen.add(sid)
        platform = entry.get("platform")
        namespace = PLATFORM_TO_NAMESPACE.get(platform)
        if namespace is None or entry.get("asw_namespace") != namespace:
            raise ValueError("transfer platform/namespace mismatch")
        if entry.get("analysis_started") is not False:
            raise ValueError("transfer entry unexpectedly started analysis")
        sample = bundle_root / str(entry.get("sample_path"))
        route_path = bundle_root / str(entry.get("route_manifest_path"))
        if sample.parent != bundle_root / namespace or route_path.parent != bundle_root / namespace:
            raise ValueError("transfer path escapes expected namespace")
        route = validate_route(sample, route_path)
        if route["submission_id"] != sid:
            raise ValueError("transfer entry/route submission mismatch")
        digest, size = hash_file(sample)
        route_digest, route_size = hash_file(route_path)
        if digest != entry.get("sha256") or size != entry.get("size"):
            raise ValueError("transfer sample manifest verification failed")
        if route_digest != entry.get("route_manifest_sha256") or route_size != entry.get("route_manifest_size"):
            raise ValueError("transfer route manifest verification failed")
    return manifest


def main() -> int:
    parser = argparse.ArgumentParser(description="Prepare or verify an explicit AmiGuard VPS -> ASW transfer bundle")
    sub = parser.add_subparsers(dest="command", required=True)
    p_prepare = sub.add_parser("prepare")
    p_prepare.add_argument("--handoff-root", required=True, type=Path)
    p_prepare.add_argument("--bundle-root", required=True, type=Path)
    p_verify = sub.add_parser("verify")
    p_verify.add_argument("--bundle-root", required=True, type=Path)
    args = parser.parse_args()
    try:
        if args.command == "prepare":
            manifest = prepare(args.handoff_root, args.bundle_root)
            print(f"PREPARED {len(manifest['entries'])} verified transfer entries -> {args.bundle_root}")
        else:
            manifest = verify(args.bundle_root)
            print(f"VERIFIED {len(manifest['entries'])} transfer entries; analysis_started=false")
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        parser.error(str(exc))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
