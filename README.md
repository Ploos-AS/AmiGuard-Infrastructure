# AmiGuard Infrastructure

Shared infrastructure contracts and deployment foundations for the AmiGuard ecosystem.

AmiGuard Infrastructure is intentionally separate from scanner/appliance code. It defines stable shared boundaries for storage, component identities, deployment assumptions, health contracts, quarantine intake and qualification gates.

## Status

M0 establishes the repository and infrastructure contract.

M1 adds the first deployable submission-service slice for a small VPS: a minimal Go service, OCI Containerfile, rootless Podman Quadlet and Caddy reverse-proxy/TLS example.

M2 adds static hardening validation and an actual rootless Podman runtime qualification in CI. The target remains 1 vCPU / 1 GiB RAM; the application container is limited to 192 MiB, runs read-only with all capabilities dropped, no-new-privileges enabled, and publishes only to loopback behind Caddy.

M3 implements the controlled quarantine protocol: explicit consent, streaming size limits, 128-bit server-generated IDs, SHA-256 while receiving, `0600` storage, atomic sample/metadata commits, and no retrieval endpoint. Client filenames are neither used as paths nor retained in the M3 receipt metadata.

M4 qualifies the real closed production VPS at `amiguard.ploos.no`: hardened SSH/firewall, DNS/TLS/Caddy, rootless Podman reboot persistence, dedicated 4 GiB `nodev,nosuid,noexec` quarantine storage, rootless UID mapping, writable quarantine bind-mount, and a controlled local end-to-end upload/hash/metadata test.

M5 begins public-intake abuse hardening in the application: upload attempts are rate-limited per client, simultaneous uploads are globally bounded, and forwarded client addresses are trusted only when the immediate peer is loopback Caddy. The existing 16 MiB sample bound, finite HTTP timeouts, consent requirement and strict multipart contract remain in force.

M5.2 adds `amiguard-admin`, a local/SSH-only quarantine CLI for metadata dashboard/list/show, sample hash verification and controlled no-overwrite export. No web-admin or retrieval endpoint is introduced.

M5.3 defines the initial production operations policy for retention/deletion, hostile-content backup handling, verified export, log/privacy hygiene, emergency upload shutdown and the final public go-live gate.

**Public sample intake remains disabled.** The production Quadlet explicitly keeps `AMIGUARD_UPLOAD_ENABLED=false`, and public POST requests to the submission endpoint must remain unavailable until the final public-intake qualification gate passes.

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
   |
/data/quarantine
   |
4 GiB dedicated quarantine filesystem on host
```

See `docs/M1_ROOTLESS_SUBMIT_SLICE.md`, `docs/M2_ROOTLESS_RUNTIME_HARDENING.md`, `docs/M3_QUARANTINE_PROTOCOL.md`, `docs/M4_PRODUCTION_VPS_QUALIFICATION.md`, `docs/M5_EDGE_ABUSE_HARDENING.md`, `docs/M5_ADMIN_CLI.md` and `docs/M5_3_PUBLIC_INTAKE_OPERATIONS.md`.

## Next milestone

The current M5 submit image and M5.2 admin CLI have been production-qualified while public uploads remain disabled. The remaining launch-critical work is to replace the enabled landing-page qualification text with production submission/consent content, review the live Caddy/journald logging configuration, perform the final public-intake qualification, and only then enable uploads.

## License

MIT. See `LICENSE`.
