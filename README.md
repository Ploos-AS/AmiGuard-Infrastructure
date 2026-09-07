# AmiGuard Infrastructure

Shared infrastructure contracts and deployment foundations for the AmiGuard ecosystem.

AmiGuard Infrastructure is intentionally separate from scanner/appliance code. It defines stable shared boundaries for storage, component identities, deployment assumptions, health contracts, quarantine intake and qualification gates.

## Status

M0 establishes the repository and infrastructure contract.

M1 adds the first deployable submission-service slice for a small VPS: a minimal Go service, OCI Containerfile, rootless Podman Quadlet and Caddy reverse-proxy/TLS example.

M2 adds static hardening validation and an actual rootless Podman runtime qualification in CI. The target remains 1 vCPU / 1 GiB RAM; the application container is limited to 192 MiB, runs read-only with all capabilities dropped, no-new-privileges enabled, and publishes only to loopback behind Caddy.

M3 implements the controlled quarantine protocol: explicit consent, streaming size limits, 128-bit server-generated IDs, SHA-256 while receiving, `0600` storage, atomic sample/metadata commits, and no retrieval endpoint. Client filenames are neither used as paths nor retained in the M3 receipt metadata.

**Public sample intake remains disabled.** The normal deployment profile still has `AMIGUARD_UPLOAD_ENABLED=false` and no writable quarantine mount. The M3 upload-enabled Quadlet is a controlled qualification example only; it must not be exposed publicly until the actual VPS passes the production hardening gate.

## Validation

Requires Python 3.11+ and Go 1.23+.

```sh
make check
```

With Podman and curl installed, run the rootless runtime gate as a normal non-root user:

```sh
make rootless-qualify
```

Optional local image build:

```sh
make container-build
```

## Scope

This repository owns shared infrastructure definitions and deployment contracts. It does not own malware samples, AmiGuard scanner implementation, historical antivirus engines, signature research conclusions, or the end-user appliance UI. Samples must never be committed here.

When hostile content is eventually accepted in production, it must live only in dedicated quarantine storage outside Git and must never be executed, extracted or served by the intake service.

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

See `docs/M1_ROOTLESS_SUBMIT_SLICE.md`, `docs/M2_ROOTLESS_RUNTIME_HARDENING.md` and `docs/M3_QUARANTINE_PROTOCOL.md`.

## Next milestone

M4 should qualify the real VPS host and public edge before uploads are enabled there: SSH/firewall, DNS/TLS, Caddy limits/rate controls, rootless ownership, quarantine filesystem policy, disk exhaustion controls, retention/deletion, backup handling, manual export with hash verification, egress restrictions and incident recovery.

## License

MIT. See `LICENSE`.
