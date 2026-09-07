#!/usr/bin/env python3
import json
import sys
from pathlib import Path

REQUIRED_COMPONENT_KEYS = {"id", "purpose", "network_exposure", "persistent_paths", "healthcheck"}
ALLOWED_EXPOSURE = {"public", "internal", "none"}

def fail(message: str) -> None:
    raise SystemExit(f"validation error: {message}")

def main() -> None:
    if len(sys.argv) != 2:
        fail("usage: validate_components.py <manifest.json>")
    path = Path(sys.argv[1])
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        fail(str(exc))
    if data.get("schema_version") != 1:
        fail("schema_version must be 1")
    if data.get("project") != "AmiGuard-Infrastructure":
        fail("project must be AmiGuard-Infrastructure")
    components = data.get("components")
    if not isinstance(components, list) or not components:
        fail("components must be a non-empty list")
    seen = set()
    for index, component in enumerate(components):
        if not isinstance(component, dict):
            fail(f"component {index} must be an object")
        missing = REQUIRED_COMPONENT_KEYS - component.keys()
        if missing:
            fail(f"component {index} missing keys: {', '.join(sorted(missing))}")
        cid = component["id"]
        if not isinstance(cid, str) or not cid or cid in seen:
            fail(f"component {index} has invalid or duplicate id")
        seen.add(cid)
        if component["network_exposure"] not in ALLOWED_EXPOSURE:
            fail(f"component {cid} has invalid network_exposure")
        paths = component["persistent_paths"]
        if not isinstance(paths, list) or not paths:
            fail(f"component {cid} must declare persistent_paths")
        for persistent_path in paths:
            if not isinstance(persistent_path, str) or not persistent_path.startswith("/data/amiguard/"):
                fail(f"component {cid} persistent path must be under /data/amiguard")
        healthcheck = component["healthcheck"]
        if not isinstance(healthcheck, str) or not healthcheck.startswith("/"):
            fail(f"component {cid} healthcheck must be an absolute HTTP path")
    print(f"OK: {len(components)} infrastructure components validated")

if __name__ == "__main__":
    main()
