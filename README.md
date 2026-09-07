# AmiGuard Infrastructure

Shared infrastructure contracts and deployment foundations for the AmiGuard ecosystem.

AmiGuard Infrastructure is intentionally separate from scanner/appliance code. It defines stable shared boundaries for storage, component identities, deployment assumptions, health contracts, and qualification gates.

## Status

M0 establishes the repository and infrastructure contract.

M1 adds the first deployable submission-service slice for a small VPS: a minimal Go service, OCI Containerfile, rootless Podman Quadlet and Caddy reverse-proxy/TLS example. The service exposes `/healthz` and a static status page only. **Sample uploads remain disabled until a later hardening milestone.**

The target VPS class is deliberately small: 1 vCPU and 1 GiB RAM. The rootless application container is limited to 192 MiB and binds only to loopback behind Caddy.

## Validation

Requires Python 3.11+ and Go 1.23+.

```sh
make check
```

Optional local container build:

```sh
make container-build
```

## Scope

This repository owns shared infrastructure definitions and deployment contracts. It does not own malware samples, AmiGuard scanner implementation, historical antivirus engines, signature research conclusions, or the end-user appliance UI. Samples must never be committed here.

Runtime hostile content, when enabled in a future milestone, must live only in dedicated quarantine storage outside Git and must never be executed by the intake service.

## Deployment model

```text
Internet
   |
 HTTPS
   |
 Caddy (host)
   |
127.0.0.1:8080
   |
rootless Podman
   |
amiguard-submit
```

See `docs/M1_ROOTLESS_SUBMIT_SLICE.md` for the M1 contract.

## Next milestone

M2 should harden the VPS deployment and qualify the rootless runtime before any upload endpoint is implemented or enabled.

## License

MIT. See `LICENSE`.
