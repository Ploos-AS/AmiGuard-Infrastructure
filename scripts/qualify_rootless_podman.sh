#!/bin/sh
set -eu

if [ "$(id -u)" -eq 0 ]; then
  echo "qualification error: must run as a non-root user" >&2
  exit 2
fi

command -v podman >/dev/null 2>&1 || {
  echo "qualification error: podman is required" >&2
  exit 2
}
command -v curl >/dev/null 2>&1 || {
  echo "qualification error: curl is required" >&2
  exit 2
}

rootless=$(podman info --format '{{.Host.Security.Rootless}}')
[ "$rootless" = "true" ] || {
  echo "qualification error: podman is not running rootless" >&2
  exit 1
}

image=localhost/amiguard-submit:m2-qualification
name=amiguard-submit-m2
port=18080

cleanup() {
  podman rm -f "$name" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

podman build -t "$image" -f submit/Containerfile submit

podman run -d \
  --name "$name" \
  --read-only \
  --cap-drop=all \
  --security-opt=no-new-privileges \
  --memory=192m \
  --pids-limit=64 \
  -e AMIGUARD_SUBMIT_ADDR=0.0.0.0:8080 \
  -p "127.0.0.1:${port}:8080" \
  "$image" >/dev/null

i=0
while [ "$i" -lt 30 ]; do
  if curl -fsS "http://127.0.0.1:${port}/healthz" >/tmp/amiguard-health.json; then
    break
  fi
  i=$((i + 1))
  sleep 1
done

curl -fsS "http://127.0.0.1:${port}/healthz" | grep -q '"status":"ok"'
curl -fsS "http://127.0.0.1:${port}/" | grep -q 'Uploads are disabled'

podman inspect "$name" --format '{{.HostConfig.ReadonlyRootfs}}' | grep -qx 'true'
podman inspect "$name" --format '{{.HostConfig.PidsLimit}}' | grep -qx '64'
podman inspect "$name" --format '{{.Config.User}}' | grep -Eq '^(65532|65532:65532)$'

# Podman versions do not serialize HostConfig.CapDrop identically. Prove the
# runtime property instead: the process must have no effective capabilities.
# Depending on Podman/version, `podman top capeff` reports either `none` or a
# zero hexadecimal mask for that state.
capeff=$(podman top "$name" capeff | awk 'NR == 2 {print $1}')
case "$capeff" in
  none|NONE|0|0000000000000000)
    ;;
  *)
    echo "qualification error: effective capabilities are not fully dropped (CapEff=$capeff)" >&2
    exit 1
    ;;
esac

echo "OK: rootless Podman runtime qualification passed"
