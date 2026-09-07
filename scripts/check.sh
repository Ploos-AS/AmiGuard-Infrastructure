#!/bin/sh
set -eu
python3 scripts/validate_components.py infrastructure/components.json
python3 -m compileall -q scripts
echo "OK: M0 checks passed"
