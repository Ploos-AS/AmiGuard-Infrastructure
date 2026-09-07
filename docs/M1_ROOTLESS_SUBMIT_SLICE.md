# M1 — Rootless Podman submit slice

M1 establishes the first deployable AmiGuard submission service slice for a very small VPS (target: 1 vCPU / 1 GiB RAM).

## Runtime shape

- host OS: minimal supported Linux host
- reverse proxy/TLS: Caddy on the host
- application: `amiguard-submit` in rootless Podman
- application bind: loopback only (`127.0.0.1:8080`)
- public exposure: Caddy only
- container memory limit: 192 MiB
- read-only container root filesystem
- all Linux capabilities dropped
- no-new-privileges enabled
- no Docker/Podman socket mounted
- no host networking

## M1 behavior

The Go service exposes:

- `GET /healthz` — machine-readable health response
- `GET /` — static status page

Actual sample upload is deliberately disabled in M1. No handler accepts uploaded sample bytes and no quarantine directory is mounted into the container yet.

## Security boundary

M1 is intentionally incapable of receiving malware. This gives us a deployable target for VPS, DNS, TLS, resource usage and rootless Podman qualification before introducing hostile input.

The future upload milestone must add its own gate covering at least size limits, atomic quarantine writes, SHA-256 calculation, immutable submission identifiers, metadata separation, rate limiting, storage permissions and non-execution guarantees.

## Local build

```sh
cd submit
go test ./...
podman build -t localhost/amiguard-submit:local -f Containerfile .
```

For rootless deployment, copy `deploy/quadlet/amiguard-submit.container` to the rootless user's `~/.config/containers/systemd/`, then reload the user systemd manager. Replace the example Caddy hostname before any public deployment.

## Qualification

M1 code qualification requires:

1. `make check` passes;
2. Go tests pass;
3. the container builds successfully;
4. the Quadlet remains loopback-only and rootless-compatible;
5. `/healthz` answers through the intended reverse-proxy path during VPS qualification;
6. the root page states uploads are disabled.

No malware samples are required or permitted for M1 qualification.
